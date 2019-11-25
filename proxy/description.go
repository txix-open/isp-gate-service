package proxy

import (
	"github.com/integration-system/isp-lib/structure"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/proxy/grpc"
	"isp-gate-service/proxy/health_check"
	"isp-gate-service/proxy/http"
	"strings"
)

var store = make(map[string]Proxy)

const (
	httpProtocol        = "http"
	grpcProtocol        = "grpc"
	healthCheckProtocol = "healthсheck"
)

type (
	Proxy interface {
		ProxyRequest(ctx *fasthttp.RequestCtx, path string) domain.ProxyResponse
		Consumer([]structure.AddressConfiguration) bool
		SkipAuth() bool
		Close()
	}
)

func Init(location conf.Location) (Proxy, error) {
	if location.PathPrefix[0] != '/' {
		return nil, errors.Errorf("path must begin with '/' in path '%s'", location.PathPrefix)
	}
	switch location.Protocol {
	case httpProtocol:
		proxy := http.NewProxy(location.SkipAuth)
		store[location.PathPrefix] = proxy
		return proxy, nil
	case grpcProtocol:
		proxy := grpc.NewProxy(location.SkipAuth)
		store[location.PathPrefix] = proxy
		return proxy, nil
	case healthCheckProtocol:
		proxy := health_check.NewProxy(location.SkipAuth)
		store[location.PathPrefix] = proxy
		return proxy, nil
	default:
		return nil, errors.Errorf("unknown protocol '%s'", location.Protocol)
	}
}

func Find(path string) (Proxy, string) {
	for pathPrefix, proxy := range store {
		if strings.HasPrefix(path, pathPrefix) {
			return proxy, getPathWithoutPrefix(path)
		}
	}
	return nil, ""
}

func Close() {
	for _, p := range store {
		p.Close()
	}
}

func getPathWithoutPrefix(path string) string {
	firstFound := false
	for i, value := range path {
		if value == '/' {
			if firstFound {
				return path[i+1:]
			} else {
				firstFound = true
			}
		}
	}
	return ""
}
