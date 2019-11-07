package handlers

import (
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/config"
	isp "github.com/integration-system/isp-lib/proto/stubs"
	s "github.com/integration-system/isp-lib/streaming"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/log_code"
	"isp-gate-service/proxy/response"
	"mime/multipart"
	"strings"
)

var sendMultipartData sendMultipartDataDesc

type sendMultipartDataDesc struct{}

func (h sendMultipartDataDesc) Complete(ctx *fasthttp.RequestCtx, method string, client *backend.RxGrpcClient) domain.ProxyResponse {
	cfg := config.GetRemote().(*conf.RemoteConfig).GrpcSetting
	timeout := cfg.GetStreamInvokeTimeout()
	bufferSize := cfg.GetTransferFileBufferSize()

	stream, cancel, err := openStream(&ctx.Request.Header, method, timeout, client)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()
	if err != nil {
		logHandlerError(log_code.TypeData.SendMultipart, method, err)
		return response.Create(
			ctx,
			response.Option.SetAndSendError(errorMsgInternal, codes.Internal, err),
			response.Option.EmptyRequest(),
			response.Option.EmptyResponse(),
		)
	}

	form, err := ctx.MultipartForm()
	defer func() {
		if form != nil {
			_ = form.RemoveAll()
		}
	}()

	if err != nil {
		logHandlerError(log_code.TypeData.SendMultipart, method, err)
		return response.Create(
			ctx,
			response.Option.SetAndSendError(errorMsgInvalidArg, codes.InvalidArgument, err),
			response.Option.EmptyRequest(),
			response.Option.EmptyResponse(),
		)
	}

	formData := make(map[string]interface{}, len(form.Value))

	for k, v := range form.Value {
		if len(v) > 0 {
			formData[k] = v[0]
		}
	}

	var (
		proxyResp domain.ProxyResponse
		resp      = make([]string, 0)
		buffer    = make([]byte, bufferSize)
		ok        = true
		eof       = false
	)

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
		if ok, eof, proxyResp = checkError(err, ctx); !ok || eof {
			break
		}

		f, err := file.Open()
		if ok, eof, proxyResp = checkError(err, ctx); !ok || eof {
			break
		}
		if ok, eof, proxyResp = h.transferFile(f, stream, buffer, ctx); ok {
			msg, err := stream.Recv()
			v, _, err := getResponse(msg, err)
			if err == nil {
				resp = append(resp, string(v))
			}
			ok = err == nil
		}

		if !ok || eof {
			break
		}
	}

	err = stream.CloseSend()
	if err != nil {
		logHandlerError(log_code.TypeData.SendMultipart, method, err)
	}

	if ok {
		arrayBody := strings.Join(resp, ",")
		_, err = ctx.WriteString("[" + arrayBody + "]")
		if err != nil {
			logHandlerError(log_code.TypeData.SendMultipart, method, err)
		}
	}
	return proxyResp
}

func (sendMultipartDataDesc) transferFile(f multipart.File, stream isp.BackendService_RequestStreamClient,
	buffer []byte, ctx *fasthttp.RequestCtx) (bool, bool, domain.ProxyResponse) {

	var (
		ok        = true
		eof       = false
		proxyResp domain.ProxyResponse
	)
	for {
		n, err := f.Read(buffer)
		if n > 0 {
			err = stream.Send(&isp.Message{Body: &isp.Message_BytesBody{BytesBody: buffer[:n]}})
			if ok, eof, proxyResp = checkError(err, ctx); !ok || eof {
				break
			}
		}
		if err != nil {
			if ok, eof, proxyResp = checkError(err, ctx); ok && eof {
				err = stream.Send(s.FileEnd())
				ok, eof, proxyResp = checkError(err, ctx)
			}
			break
		}
	}
	return ok, eof, proxyResp
}
