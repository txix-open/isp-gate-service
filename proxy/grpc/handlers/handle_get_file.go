package handlers

import (
	"fmt"
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/config"
	s "github.com/integration-system/isp-lib/streaming"
	u "github.com/integration-system/isp-lib/utils"
	log "github.com/integration-system/isp-log"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"io"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy/response"
	"strconv"
)

var getFile getFileDesc

type getFileDesc struct{}

func (g getFileDesc) Complete(ctx *fasthttp.RequestCtx, method string, client *backend.RxGrpcClient) domain.ProxyResponse {
	cfg := config.GetRemote().(*conf.RemoteConfig).GrpcSetting
	timeout := cfg.GetStreamInvokeTimeout()

	req, err := g.readJsonBody(ctx)
	if err != nil {
		logHandlerError(log_code.TypeData.GetFile, method, err)
		return response.Create(
			ctx,
			response.Option.SetAndSendError(err.Error(), codes.InvalidArgument, err),
			response.Option.EmptyRequest(),
			response.Option.EmptyResponse(),
		)
	}

	stream, cancel, err := openStream(&ctx.Request.Header, method, timeout, client)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()
	if err != nil {
		logHandlerError(log_code.TypeData.GetFile, method, err)
		return response.Create(
			ctx,
			response.Option.SetAndSendError(errorMsgInternal, codes.Internal, err),
			response.Option.EmptyRequest(),
			response.Option.EmptyResponse(),
		)
	}

	if req != nil {
		value := u.ConvertInterfaceToGrpcStruct(req)
		err := stream.Send(backend.WrapBody(value))
		if err != nil {
			logHandlerError(log_code.TypeData.GetFile, method, err)
			return response.Create(
				ctx,
				response.Option.SetAndSendError(errorMsgInternal, codes.Internal, err),
				response.Option.EmptyRequest(),
				response.Option.EmptyResponse(),
			)
		}
	}

	msg, err := stream.Recv()
	if err != nil {
		bytes, status, err := getResponse(nil, err)
		if err == nil {
			ctx.SetStatusCode(status)
			ctx.SetBody(bytes)
		}
		return response.Create(ctx, response.Option.EmptyResponse(), response.Option.EmptyRequest())
	}
	bf := s.BeginFile{}
	err = bf.FromMessage(msg)
	if err != nil {
		bytes, status, err := getResponse(nil, err)
		if err == nil {
			ctx.SetStatusCode(status)
			ctx.SetBody(bytes)
		}
		return response.Create(ctx, response.Option.EmptyResponse(), response.Option.EmptyRequest())
	}
	header := &ctx.Response.Header
	header.Set(headerKeyContentDisposition, fmt.Sprintf("attachment; filename=%s", bf.FileName))
	header.Set(headerKeyContentType, bf.ContentType)
	if bf.ContentLength > 0 {
		header.Set(headerKeyContentLength, strconv.Itoa(int(bf.ContentLength)))
	} else {
		header.Set(headerKeyTransferEncoding, "chunked")
	}

	for {
		msg, err := stream.Recv()
		if s.IsEndOfFile(msg) || err == io.EOF {
			break
		}
		if err != nil {
			logHandlerError(log_code.TypeData.GetFile, method, err)
			break
		}
		bytes := msg.GetBytesBody()
		if bytes == nil {
			log.WithMetadata(map[string]interface{}{
				log_code.MdTypeData: log_code.TypeData.GetFile,
				log_code.MdMethod:   method,
			}).Errorf(log_code.WarnProxyGrpcHandler, "Method %s. Expected bytes array", method)
			break
		}
		_, err = ctx.Write(bytes)
		if err != nil {
			logHandlerError(log_code.TypeData.GetFile, method, err)
			break
		}
	}
	return response.Create(ctx, response.Option.EmptyResponse(), response.Option.EmptyRequest())
}

func (getFileDesc) readJsonBody(ctx *fasthttp.RequestCtx) (interface{}, error) {
	requestBody := ctx.Request.Body()
	var body interface{}
	if len(requestBody) == 0 {
		requestBody = []byte("{}")
	}
	if requestBody[0] == '{' {
		body = make(map[string]interface{})
	} else if requestBody[0] == '[' {
		body = make([]interface{}, 0)
	} else {
		return nil, errors.New("Invalid json format. Expected object or array")
	}

	err := json.Unmarshal(requestBody, &body)

	if err != nil {
		return nil, errors.New("Not able to read request body")
	}
	return body, err
}
