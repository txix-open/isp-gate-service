package proxy

import (
	"sort"
	"strings"

	"github.com/integration-system/isp-lib/v2/structure"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"isp-gate-service/domain"
	"isp-gate-service/proxy/grpc"
	"isp-gate-service/proxy/health_check"
	"isp-gate-service/proxy/http"
	"isp-gate-service/proxy/websocket"
)

var store = make([]storeItem, 0)

const (
	httpProtocol        = "http"
	websocketProtocol   = "websocket"
	grpcProtocol        = "grpc"
	healthCheckProtocol = "healthсheck"
)

type (
	Proxy interface {
		ProxyRequest(ctx *fasthttp.RequestCtx, path string) domain.ProxyResponse
		Consumer([]structure.AddressConfiguration) bool
		SkipAuth() bool
		SkipExistCheck() bool
		Close()
	}
	storeItem struct {
		pathPrefix string
		proxy      Proxy
	}
)

func Init(protocol, pathPrefix string, skipAuth, skipExistCheck bool) (Proxy, error) {
	if pathPrefix[0] != '/' {
		return nil, errors.Errorf("path must begin with '/' in path '%s'", pathPrefix)
	}
	var proxy Proxy
	switch protocol {
	case httpProtocol:
		proxy = http.NewProxy(skipAuth, skipExistCheck)
	case grpcProtocol:
		proxy = grpc.NewProxy(skipAuth, skipExistCheck)
	case healthCheckProtocol:
		proxy = health_check.NewProxy(skipAuth, skipExistCheck)
	case websocketProtocol:
		proxy = websocket.NewProxy(skipAuth, skipExistCheck)
	default:
		return nil, errors.Errorf("unknown protocol '%s'", protocol)
	}

	store = append(store, storeItem{
		pathPrefix: pathPrefix,
		proxy:      proxy,
	})
	return proxy, nil
}

func PostInit() {
	sort.Slice(store, func(i, j int) bool {
		return store[i].pathPrefix > store[j].pathPrefix
	})
}

func Find(path string) (Proxy, string) {
	for _, item := range store {
		if strings.HasPrefix(path, item.pathPrefix) {
			return item.proxy, getPathWithoutPrefix(path, item.pathPrefix)
		}
	}
	return nil, ""
}

func Close() {
	for _, p := range store {
		p.proxy.Close()
	}
}

func getPathWithoutPrefix(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	return s
}
