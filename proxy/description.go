package proxy

import (
	"github.com/integration-system/isp-lib/structure"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"isp-gate-service/conf"
	"isp-gate-service/proxy/grpc"
	"isp-gate-service/proxy/http"
	"strings"
)

var store = make(map[string]Proxy)

const (
	httpProtocol = "http"
	grpcProtocol = "grpc"
)

type Proxy interface {
	ProxyRequest(ctx *fasthttp.RequestCtx)
	Consumer([]structure.AddressConfiguration) bool
	Close()
}

func Init(location conf.Location) (Proxy, error) {
	if location.PathPrefix[0] != '/' {
		return nil, errors.Errorf("path must begin with '/' in path '%s'", location.PathPrefix)
	}
	switch location.Protocol {
	case httpProtocol:
		proxy := http.NewProxy()
		store[location.PathPrefix] = proxy
		return proxy, nil
	case grpcProtocol:
		proxy := grpc.NewProxy()
		store[location.PathPrefix] = proxy
		return proxy, nil
	default:
		return nil, errors.New("unknown protocol")
	}
}

func Find(path string) Proxy {
	for pathPrefix, proxy := range store {
		if strings.HasPrefix(path, pathPrefix) {
			return proxy
		}
	}
	return nil
}

func Close() {
	for _, p := range store {
		p.Close()
	}
}
