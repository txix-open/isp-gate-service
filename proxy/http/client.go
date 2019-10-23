package http

import (
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"github.com/valyala/fasthttp"
	"isp-gate-service/log_code"
	"net"
	"strings"
)

type httpProxy struct {
	client *fasthttp.HostClient
}

func NewProxy() *httpProxy {
	return &httpProxy{client: nil}
}

func (p *httpProxy) Consumer(addressList []structure.AddressConfiguration) bool {
	addresses := make([]string, len(addressList))
	for key, addr := range addressList {
		addresses[key] = addr.GetAddress()
	}

	p.client = &fasthttp.HostClient{
		Addr: strings.Join(addresses, `,`),
	}
	return true
}

func (p *httpProxy) ProxyRequest(ctx *fasthttp.RequestCtx) {
	req := &ctx.Request
	res := &ctx.Response

	if addr, _, err := net.SplitHostPort(ctx.RemoteAddr().String()); err == nil {
		req.Header.Add("X-Forwarded-For", addr)
	}

	if err := p.client.Do(req, res); err != nil {
		log.Error(log_code.ErrorClientHttp, err)
		return
	}
}

func (p *httpProxy) Close() {
	p.client = nil
}
