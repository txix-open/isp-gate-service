package grpc

import (
	"github.com/integration-system/isp-lib/backend"
	u "github.com/integration-system/isp-lib/utils"
	"github.com/valyala/fasthttp"
	"isp-gate-service/proxy/grpc/handlers"
	"isp-gate-service/proxy/grpc/utils"
	"mime"
)

type Handler interface {
	Complete(ctx *fasthttp.RequestCtx, method string, client *backend.RxGrpcClient)
}

func SetHandler(ctx *fasthttp.RequestCtx) Handler {
	isMultipart := isMultipart(ctx)
	isExpectFile := string(ctx.Request.Header.Peek(u.ExpectFileHeader)) == "true"

	if isMultipart {
		ctx.Response.Header.SetContentType(utils.JsonContentType)
		return handlers.SendMultipartData
	} else if isExpectFile {
		return handlers.GetFile
	} else {
		ctx.Response.Header.SetContentType(utils.JsonContentType)
		return handlers.Json
	}
}

func isMultipart(ctx *fasthttp.RequestCtx) bool {
	if !ctx.IsPost() {
		return false
	}
	v := string(ctx.Request.Header.ContentType())
	if v == "" {
		return false
	}
	d, params, err := mime.ParseMediaType(v)
	if err != nil || d != "multipart/form-data" {
		return false
	}
	_, ok := params["boundary"]
	if !ok {
		return false
	}
	return true
}
