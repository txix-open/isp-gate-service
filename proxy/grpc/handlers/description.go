package handlers

import (
	"github.com/integration-system/isp-lib/v2/backend"
	"github.com/valyala/fasthttp"
	"isp-gate-service/domain"
	"isp-gate-service/utils"
)

var Handler handlerHelper

type (
	handlerHelper struct{}

	handler interface {
		Complete(ctx *fasthttp.RequestCtx, method string, client *backend.RxGrpcClient) domain.ProxyResponse
	}
)

func (h handlerHelper) Get(ctx *fasthttp.RequestCtx) handler {
	ctx.Response.Header.SetContentTypeBytes(utils.JsonContentType)
	return handleJson
}
