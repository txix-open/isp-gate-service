package handlers

import (
	"fmt"
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/config"
	s "github.com/integration-system/isp-lib/streaming"
	u "github.com/integration-system/isp-lib/utils"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"io"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy/grpc/utils"
	"strconv"
)

var GetFile getFile

type getFile struct{}

func (getFile) Complete(ctx *fasthttp.RequestCtx, method string, client *backend.RxGrpcClient) {
	cfg := config.GetRemote().(*conf.RemoteConfig)
	timeout := cfg.GetStreamInvokeTimeout()

	req, err := utils.ReadJsonBody(ctx)
	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.GetFile, method, err)
		utils.SendError(err.Error(), codes.InvalidArgument, nil, ctx)
		return
	}

	stream, cancel, err := openStream(&ctx.Request.Header, method, timeout, client)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()
	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.GetFile, method, err)
		utils.SendError(errorMsgInternal, codes.Internal, []interface{}{err.Error()}, ctx)
		return
	}

	if req != nil {
		value := u.ConvertInterfaceToGrpcStruct(req)
		err := stream.Send(backend.WrapBody(value))
		if err != nil {
			utils.LogRequestHandlerError(log_code.TypeData.GetFile, method, err)
			utils.SendError(errorMsgInternal, codes.Internal, []interface{}{err.Error()}, ctx)
			return
		}
	}

	msg, err := stream.Recv()
	if err != nil {
		bytes, status, err := utils.GetResponse(nil, err)
		if err == nil {
			ctx.SetStatusCode(status)
			ctx.SetBody(bytes)
		}
		return
	}
	bf := s.BeginFile{}
	err = bf.FromMessage(msg)
	if err != nil {
		bytes, status, err := utils.GetResponse(nil, err)
		if err == nil {
			ctx.SetStatusCode(status)
			ctx.SetBody(bytes)
		}
		return
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
			utils.LogRequestHandlerError(log_code.TypeData.GetFile, method, err)
			break
		}
		bytes := msg.GetBytesBody()
		if bytes == nil {
			log.WithMetadata(map[string]interface{}{
				log_code.MdTypeData: log_code.TypeData.GetFile,
				log_code.MdMethod:   method,
			}).Errorf(log_code.WarnRequestHandler, "Method %s. Expected bytes array", method)
			break
		}
		_, err = ctx.Write(bytes)
		if err != nil {
			utils.LogRequestHandlerError(log_code.TypeData.GetFile, method, err)
			break
		}
	}
}
