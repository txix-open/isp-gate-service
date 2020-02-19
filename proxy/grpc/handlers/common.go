package handlers

import (
	"github.com/golang/protobuf/proto"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/config"
	http2 "github.com/integration-system/isp-lib/http"
	isp "github.com/integration-system/isp-lib/proto/stubs"
	"github.com/integration-system/isp-lib/structure"
	"github.com/integration-system/isp-lib/utils"
	log "github.com/integration-system/isp-log"
	jsoniter "github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/log_code"
	utils2 "isp-gate-service/utils"
	"net/http"
	"strings"
	"time"
)

const (
	headerKeyContentDisposition = "Content-Disposition"
	headerKeyContentType        = "Content-Type"
	headerKeyContentLength      = "Content-Length"
	headerKeyTransferEncoding   = "Transfer-Encoding"

	errorMsgInternal   = "Internal server error"
	errorMsgInvalidArg = "Not able to read request body"
)

var (
	json      = jsoniter.ConfigFastest
	emptyBody = make([]byte, 0)
)

func convertError(err error) ([]byte, int) {
	s, ok := status.FromError(err)
	if ok {
		cfg := config.GetRemote().(*conf.RemoteConfig).GrpcSetting
		if cfg.EnableOriginalProtoErrors {
			if body, err := proto.Marshal(s.Proto()); err != nil {
				return []byte(utils.ServiceError), http.StatusServiceUnavailable
			} else {
				return body, http2.CodeToHttpStatus(s.Code())
			}
		} else {
			details := s.Details()
			newDetails := make([]interface{}, len(details))
			for i, detail := range details {
				switch typeOfDetail := detail.(type) {
				case *structpb.Struct:
					newDetails[i] = utils.ConvertGrpcStructToInterface(
						&structpb.Value{Kind: &structpb.Value_StructValue{StructValue: typeOfDetail}},
					)
				case *isp.Message:
					newDetails[i] = utils.ConvertGrpcStructToInterface(
						backend.ResolveBody(typeOfDetail),
					)
				default:
					newDetails[i] = typeOfDetail
				}
			}

			var respBody interface{}
			if cfg.ProxyGrpcErrorDetails && len(newDetails) > 0 {
				respBody = newDetails[0]
			} else {
				respBody = structure.GrpcError{ErrorMessage: s.Message(), ErrorCode: s.Code().String(), Details: newDetails}
			}
			if errorData, err := json.Marshal(respBody); err != nil {
				log.Warn(log_code.WarnConvertErrorDataMarshalResponse, err)
				return []byte(utils.ServiceError), http.StatusServiceUnavailable
			} else {
				return errorData, http2.CodeToHttpStatus(s.Code())
			}
		}
	} else {
		return []byte(utils.ServiceError), http.StatusServiceUnavailable
	}
}

func getResponse(msg *isp.Message, err error) ([]byte, int, error) {
	if err != nil {
		errorBody, errorStatus := convertError(err)
		return errorBody, errorStatus, nil
	}

	bytes := msg.GetBytesBody()
	if bytes != nil {
		return bytes, http.StatusOK, nil
	}
	result := backend.ResolveBody(msg)
	data := utils.ConvertGrpcStructToInterface(result)
	byteResponse, err := json.Marshal(data)
	return byteResponse, http.StatusOK, err
}

func logHandlerError(typeData, method string, err error) {
	log.WithMetadata(map[string]interface{}{
		log_code.MdTypeData: typeData,
		log_code.MdMethod:   method,
	}).Warn(log_code.WarnProxyGrpcHandler, err)
}

func openStream(headers *fasthttp.RequestHeader, method string, timeout time.Duration, client *backend.RxGrpcClient) (
	isp.BackendService_RequestStreamClient, context.CancelFunc, error) {

	cli, err := client.Conn()
	if err != nil {
		return nil, nil, err
	}
	md, _ := makeMetadata(headers, method)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	stream, err := cli.RequestStream(ctx)
	if err != nil {
		return nil, nil, err
	}
	return stream, cancel, nil
}

func makeMetadata(r *fasthttp.RequestHeader, method string) (metadata.MD, string) {
	//method = strings.TrimPrefix(method, "/api/")
	md := metadata.Pairs(utils.ProxyMethodNameHeader, method)
	r.VisitAll(func(key, v []byte) {
		lowerHeader := strings.ToLower(string(key))
		if len(v) > 0 && strings.HasPrefix(lowerHeader, "x-") {
			md = metadata.Join(md, metadata.Pairs(lowerHeader, string(v)))
		}
	})
	return md, method
}

func checkError(err error, ctx *fasthttp.RequestCtx, method string) (bool, bool, domain.ProxyResponse) {
	var (
		ok                    = true
		eof                   = false
		resp                  = domain.ProxyResponse{}
		msg                   = errorMsgInternal
		code                  = codes.Internal
		details []interface{} = nil
	)

	if err != nil {
		if err != io.EOF {
			s, itStatus := status.FromError(err)
			if itStatus {
				msg = s.Message()
				code = s.Code()
				details = s.Details()
			}
			logHandlerError(log_code.TypeData.GetFile, method, err)
			utils2.WriteError(ctx, msg, code, details)
			resp = domain.Create().SetError(err)
			ok, eof = false, false
		} else {
			ok, eof = true, true
		}
	}
	return ok, eof, resp
}
