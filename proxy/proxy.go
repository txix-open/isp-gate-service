package proxy

import (
	"strings"

	"github.com/integration-system/isp-lib/v2/structure"
	"github.com/valyala/fasthttp"
	"isp-gate-service/domain"
	"isp-gate-service/proxy/http"
)

var (
	RoutingProxy  *http.HttpProxy
	ApiPathPrefix string
)

type (
	Proxy interface {
		ProxyRequest(ctx *fasthttp.RequestCtx, path string) domain.ProxyResponse
		Consumer([]structure.AddressConfiguration) bool
		SkipAuth() bool
		SkipExistCheck() bool
		Close()
	}
)

func InitRoutingProxy(skipAuth bool, skipExistCheck bool) func(list []structure.AddressConfiguration) bool {
	RoutingProxy = http.NewProxy(skipAuth, skipExistCheck)
	return func(list []structure.AddressConfiguration) bool {
		RoutingProxy.Consumer(list)
		return true
	}
}

func GetPathWithoutApiPrefix(initialPath string) string {
	s := strings.TrimPrefix(initialPath, ApiPathPrefix)
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	return s
}

func Close() {
	RoutingProxy.Close()
}
