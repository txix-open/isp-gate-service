package health_check

import (
	"github.com/integration-system/isp-lib/structure"
	"github.com/valyala/fasthttp"
	"isp-gate-service/domain"
)

type healthCheckProxy struct {
}

func NewProxy() *healthCheckProxy {
	return &healthCheckProxy{}
}

func (p *healthCheckProxy) Consumer(addressList []structure.AddressConfiguration) bool {
	return true
}

func (p *healthCheckProxy) ProxyRequest(ctx *fasthttp.RequestCtx) domain.ProxyResponse {
	return domain.Create()
}

func (p *healthCheckProxy) Close() {

}
