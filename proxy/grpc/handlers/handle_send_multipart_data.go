package handlers

import (
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/config"
	s "github.com/integration-system/isp-lib/streaming"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/conf"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy/grpc/utils"
	"strings"
)

var SendMultipartData sendMultipartData

type sendMultipartData struct{}

func (sendMultipartData) Complete(ctx *fasthttp.RequestCtx, method string, client *backend.RxGrpcClient) {
	cfg := config.GetRemote().(*conf.RemoteConfig)
	timeout := cfg.GetStreamInvokeTimeout()
	bufferSize := cfg.GetTransferFileBufferSize()

	stream, cancel, err := openStream(&ctx.Request.Header, method, timeout, client)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()
	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.SendMultipart, method, err)
		utils.SendError(errorMsgInternal, codes.Internal, []interface{}{err.Error()}, ctx)
		return
	}

	form, err := ctx.MultipartForm()
	defer func() {
		if form != nil {
			_ = form.RemoveAll()
		}
	}()

	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.SendMultipart, method, err)
		utils.SendError(errorMsgInvalidArg, codes.InvalidArgument, []interface{}{err.Error()}, ctx)
		return
	}

	formData := make(map[string]interface{}, len(form.Value))

	for k, v := range form.Value {
		if len(v) > 0 {
			formData[k] = v[0]
		}
	}

	response := make([]string, 0)
	buffer := make([]byte, bufferSize)
	ok := true
	eof := false

	for formDataName, files := range form.File {
		if len(files) == 0 {
			continue
		}
		file := files[0]
		fileName := file.Filename
		contentType := file.Header.Get(headerKeyContentType)
		contentLength := file.Size
		bf := s.BeginFile{
			FileName:      fileName,
			FormDataName:  formDataName,
			ContentType:   contentType,
			ContentLength: contentLength,
			FormData:      formData,
		}
		err = stream.Send(bf.ToMessage())
		if ok, eof = checkError(err, ctx); !ok || eof {
			break
		}

		f, err := file.Open()
		if ok, eof = checkError(err, ctx); !ok || eof {
			break
		}
		if ok, eof = transferFile(f, stream, buffer, ctx); ok {
			msg, err := stream.Recv()
			v, _, err := utils.GetResponse(msg, err)
			if err == nil {
				response = append(response, string(v))
			}
			ok = err == nil
		}

		if !ok || eof {
			break
		}
	}

	err = stream.CloseSend()
	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.SendMultipart, method, err)
	}

	if ok {
		arrayBody := strings.Join(response, ",")
		_, err = ctx.WriteString("[" + arrayBody + "]")
		if err != nil {
			utils.LogRequestHandlerError(log_code.TypeData.SendMultipart, method, err)
		}
	}
}
