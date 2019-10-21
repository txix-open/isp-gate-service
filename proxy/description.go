package proxy

import (
	"errors"
	"github.com/integration-system/isp-lib/structure"
	"github.com/valyala/fasthttp"
	"isp-gate-service/conf"
	"isp-gate-service/proxy/grpc"
	"isp-gate-service/proxy/http"
)

var ProxyStore = make(map[string]Proxy)

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
	switch location.Protocol {
	case httpProtocol:
		proxy := http.NewProxy()
		ProxyStore[location.PathPrefix] = proxy
		return proxy, nil
	case grpcProtocol:
		proxy := grpc.NewProxy()
		ProxyStore[location.PathPrefix] = proxy
		return proxy, nil
	default:
		return nil, errors.New("unknown protocol")
	}
}
