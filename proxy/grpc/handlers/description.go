package handlers

import (
	"github.com/integration-system/isp-lib/backend"
	u "github.com/integration-system/isp-lib/utils"
	"github.com/valyala/fasthttp"
	"isp-gate-service/utils"
	"mime"
)

var Handler handlerHelper

type (
	handlerHelper struct{}

	handler interface {
		Complete(ctx *fasthttp.RequestCtx, method string, client *backend.RxGrpcClient)
	}
)

func (h handlerHelper) Get(ctx *fasthttp.RequestCtx) handler {
	isMultipart := h.isMultipart(ctx)
	isExpectFile := string(ctx.Request.Header.Peek(u.ExpectFileHeader)) == "true"

	if isMultipart {
		ctx.Response.Header.SetContentType(utils.JsonContentType)
		return sendMultipartData
	} else if isExpectFile {
		return getFile
	} else {
		ctx.Response.Header.SetContentType(utils.JsonContentType)
		return handleJson
	}
}

func (h handlerHelper) isMultipart(ctx *fasthttp.RequestCtx) bool {
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
