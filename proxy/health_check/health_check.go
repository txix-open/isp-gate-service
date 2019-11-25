package health_check

import (
	"github.com/integration-system/isp-lib/structure"
	"github.com/valyala/fasthttp"
	"isp-gate-service/domain"
)

type healthCheckProxy struct {
	skipAuth bool
}

func NewProxy(skipAuth bool) *healthCheckProxy {
	return &healthCheckProxy{skipAuth: skipAuth}
}

func (p *healthCheckProxy) Consumer(addressList []structure.AddressConfiguration) bool {
	return true
}

func (p *healthCheckProxy) ProxyRequest(ctx *fasthttp.RequestCtx, path string) domain.ProxyResponse {
	ctx.Response.SetBody(ctx.Request.Body())
	ctx.Request.SetRequestURI(path)
	return domain.Create()
}

func (p *healthCheckProxy) SkipAuth() bool {
	return p.skipAuth
}

func (p *healthCheckProxy) Close() {

}
