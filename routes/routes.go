package routes

import (
	"context"
	"isp-gate-service/domain"
	"isp-gate-service/middleware"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"

	"github.com/txix-open/isp-kit/cluster"
	"github.com/txix-open/isp-kit/log"
)

type Routes struct {
	logger         log.Logger
	allHttpMethods []string
	router         *httprouter.Router
}

func NewRoutes(logger log.Logger) *Routes {
	return &Routes{
		logger: logger,
		allHttpMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodConnect,
			http.MethodOptions,
			http.MethodTrace,
		},
		router: httprouter.New(),
	}
}

func (s *Routes) ReceiveRoutes(ctx context.Context, routes cluster.RoutingConfig) error {
	router := httprouter.New()

	for _, backend := range routes {
		for _, descriptor := range backend.Endpoints {
			path := normalizeDescriptorPath(descriptor.Path)
			if descriptor.HttpMethod != "" {
				s.registerEndpoint(router, descriptor.HttpMethod, path, descriptor)
			} else {
				for _, httpMethod := range s.allHttpMethods {
					s.registerEndpoint(router, httpMethod, path, descriptor)
				}
			}
		}
	}

	s.router = router
	return nil
}

func (s *Routes) ResolveEndpoint(method string, path string, cfg middleware.EntryPointConfig) (*domain.EndpointMeta, error) {
	lookupPath := path
	if !cfg.WithPrefix {
		lookupPath = strings.TrimPrefix(lookupPath, cfg.PathPrefix)
	}

	handler, params, _ := s.router.Lookup(method, lookupPath)

	metaEndpoint := lookupPath
	if !cfg.WithLendingSlash {
		metaEndpoint = strings.TrimLeft(metaEndpoint, "/")
	}

	if handler == nil {
		if cfg.ErrorOnUnknownEndpoint {
			return nil, errors.Errorf("unknown endpoint '%s'", lookupPath)
		}
		return &domain.EndpointMeta{
			Endpoint: metaEndpoint,
		}, nil
	}

	req, _ := http.NewRequestWithContext(context.Background(), method, lookupPath, nil)
	handler(nil, req, params)

	meta := domain.EndpointMetaFromContext(req.Context())
	meta.Endpoint = metaEndpoint
	return &meta, nil
}

func (s *Routes) registerEndpoint(
	router *httprouter.Router,
	httpMethod string,
	path string,
	descriptor cluster.EndpointDescriptor,
) {
	defer func() {
		_ = recover()
	}()

	requiredAdminPerm, _ := cluster.GetRequiredAdminPermission(descriptor)
	meta := domain.EndpointMeta{
		Inner:                   descriptor.Inner,
		RequiredAdminPermission: requiredAdminPerm,
		PathSchema:              descriptor.Path,
	}

	router.HandlerFunc(
		httpMethod,
		path,
		func(_ http.ResponseWriter, r *http.Request) {
			ctx := meta.ToContext(r.Context())
			// хак для передачи метадаты обратно в код
			*r = *r.WithContext(ctx)
		},
	)
}
