package proxy

import (
	"sort"
	"strings"

	"github.com/integration-system/isp-lib/v2/structure"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"isp-gate-service/conf"
	"isp-gate-service/domain"
	"isp-gate-service/proxy/grpc"
	"isp-gate-service/proxy/health_check"
	"isp-gate-service/proxy/http"
	"isp-gate-service/proxy/websocket"
)

var (
	store = make([]storeItem, 0)
)

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
		paths      []string // if pathPrefix is empty
		proxy      Proxy
	}
)

func InitProxies(locations []conf.Location) (map[string]func([]structure.AddressConfiguration) bool, error) {
	locationsByTargetModule := conf.GetLocationsByTargetModule(locations)
	requiredModules := make(map[string]func([]structure.AddressConfiguration) bool)

	for targetModule, locations := range locationsByTargetModule {
		consumerStorage := make([]func([]structure.AddressConfiguration) bool, len(locations))
		for i, location := range locations {
			if location.PathPrefix[0] != '/' {
				return nil, errors.Errorf("path must begin with '/' in path '%s'", location.PathPrefix)
			}
			p, err := makeProxy(location.Protocol, location.SkipAuth, location.SkipExistCheck)
			if err != nil {
				return nil, err
			}
			store = append(store, storeItem{
				pathPrefix: location.PathPrefix,
				proxy:      p,
			})
			consumerStorage[i] = p.Consumer
		}
		requiredModules[targetModule] = aggregateConsumers(consumerStorage...)
	}

	sort.Slice(store, func(i, j int) bool {
		return store[i].pathPrefix > store[j].pathPrefix
	})

	return requiredModules, nil
}

func InitProxiesFromConfigs(configs structure.RoutingConfig) error {
	for _, config := range configs { //nolint
		ip := config.Address.IP

		for protocol, info := range config.HandlersInfo {
			p, err := makeProxy(protocol, info.SkipAuth, info.SkipExistCheck)
			if err != nil {
				return errors.Wrapf(err, "bad dynamic config in service %s with protocol %s", config.ModuleName, protocol)
			}
			addressConfig := []structure.AddressConfiguration{{
				Port: info.Port,
				IP:   ip,
			}}
			p.Consumer(addressConfig)
			item := storeItem{
				pathPrefix: "",
				proxy:      p,
				paths:      getPathsFromEndpoints(info.Endpoints),
			}
			store = append(store, item)
		}
	}
	return nil
}

func getPathsFromEndpoints(endpoints []structure.EndpointDescriptor) []string {
	paths := make([]string, len(endpoints))
	for _, endpoint := range endpoints {
		paths = append(paths, endpoint.Path)
	}
	return paths
}

func makeProxy(protocol string, skipAuth, skipExistCheck bool) (Proxy, error) {
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

	return proxy, nil
}

func Find(path string) (Proxy, string) {
	for _, item := range store {
		if item.pathPrefix != "" {
			if strings.HasPrefix(path, item.pathPrefix) {
				return item.proxy, getPathWithoutPrefix(path, item.pathPrefix)
			}
		}
		for _, iPath := range item.paths {
			if path == iPath {
				return item.proxy, path
			}
			if path == "/"+iPath {
				return item.proxy, path[1:]
			}
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

func aggregateConsumers(consumers ...func([]structure.AddressConfiguration) bool) func([]structure.AddressConfiguration) bool {
	return func(list []structure.AddressConfiguration) bool {
		for _, consumer := range consumers {
			consumer(list)
		}
		return true
	}
}
