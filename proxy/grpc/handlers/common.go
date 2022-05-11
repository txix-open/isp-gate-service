package handlers

import (
	"bytes"
	"net/http"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/integration-system/isp-lib/v2/backend"
	"github.com/integration-system/isp-lib/v2/config"
	http2 "github.com/integration-system/isp-lib/v2/http"
	"github.com/integration-system/isp-lib/v2/isp"
	"github.com/integration-system/isp-lib/v2/structure"
	"github.com/integration-system/isp-lib/v2/utils"
	log "github.com/integration-system/isp-log"
	jsoniter "github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
)

const (
	errorMsgInternal = "Internal server error"
	metadataSize     = 5
)

var json = jsoniter.ConfigFastest

func convertError(err error) ([]byte, int) {
	s, ok := status.FromError(err)
	if !ok {
		return []byte(utils.ServiceError), http.StatusServiceUnavailable
	}
	cfg := config.GetRemote().(*conf.RemoteConfig).GrpcSetting
	if cfg.EnableOriginalProtoErrors {
		body, err := proto.Marshal(s.Proto())
		if err != nil {
			return []byte(utils.ServiceError), http.StatusServiceUnavailable
		}
		return body, http2.CodeToHttpStatus(s.Code())
	}

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

	errorData, err := json.Marshal(respBody)
	if err != nil {
		log.Warn(log_code.WarnConvertErrorDataMarshalResponse, err)
		return []byte(utils.ServiceError), http.StatusServiceUnavailable
	}
	return errorData, http2.CodeToHttpStatus(s.Code())
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

func makeMetadata(r *fasthttp.RequestHeader, method string) (metadata.MD, string) {
	md := make(metadata.MD, metadataSize)
	md[utils.ProxyMethodNameHeader] = []string{method}
	r.VisitAll(func(key, v []byte) {
		lowerHeader := bytes.ToLower(key)
		if len(v) > 0 && bytes.HasPrefix(lowerHeader, []byte("x-")) {
			md[string(lowerHeader)] = []string{string(v)}
		}
	})
	return md, method
}
