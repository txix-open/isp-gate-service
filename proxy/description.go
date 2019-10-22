package proxy

import (
	"github.com/integration-system/isp-lib/structure"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"isp-gate-service/conf"
	"isp-gate-service/proxy/grpc"
	"isp-gate-service/proxy/grpc/utils"
	"isp-gate-service/proxy/http"
	"strings"
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
	if location.PathPrefix[0] != '/' {
		return nil, errors.Errorf("path must begin with '/' in path '%s'", location.PathPrefix)
	}
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

func Handle(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	for pathPrefix, proxy := range ProxyStore {
		if strings.HasPrefix(path, pathPrefix) {
			proxy.ProxyRequest(ctx)
			return
		}
	}
	utils.SendError("unknown path", codes.Internal, []interface{}{map[string]string{"path": path}}, ctx)
}
