package routes

import (
	"context"
	"fmt"
	"isp-gate-service/middleware"
	"strings"

	"github.com/txix-open/isp-kit/cluster"
)

type Routes struct {
	allEndpoints        map[string]bool
	innerEndpoints      map[string]bool
	requiredPermissions map[string]string
}

func NewRoutes() *Routes {
	return &Routes{
		allEndpoints:        make(map[string]bool),
		innerEndpoints:      make(map[string]bool),
		requiredPermissions: make(map[string]string),
	}
}

func (s *Routes) ReceiveRoutes(ctx context.Context, routes cluster.RoutingConfig) error {
	newAllEndpoints := make(map[string]bool)
	newInnerMethods := make(map[string]bool)
	newRequiredPermissions := make(map[string]string)
	for _, backend := range routes {
		for _, v := range backend.Endpoints {
			newAllEndpoints[v.Path] = true
			if v.Inner {
				newInnerMethods[v.Path] = true
			}
			perm, ok := cluster.GetRequiredAdminPermission(v)
			if ok {
				newRequiredPermissions[v.Path] = perm
			}
		}
	}

	s.allEndpoints = newAllEndpoints
	s.innerEndpoints = newInnerMethods
	s.requiredPermissions = newRequiredPermissions

	return nil
}

func (s *Routes) ResolveEndpoint(path string, cfg middleware.EntryPointConfig) string {
	if cfg.WithPrefix {
		return path
	}

	prefix := fmt.Sprintf("%s/", cfg.PathPrefix)
	noPrefixPath := strings.TrimPrefix(path, prefix)
	ok := s.allEndpoints[noPrefixPath]
	if ok {
		return noPrefixPath
	}

	invertedLeadingSlash := invertLeadingSlash(noPrefixPath)
	ok = s.allEndpoints[invertedLeadingSlash]
	if ok {
		return invertedLeadingSlash
	}

	return "unknown_endpoint"
}

func (s *Routes) IsInnerEndpoint(endpoint string) bool {
	return s.innerEndpoints[endpoint]
}

func (s *Routes) RequiredAdminPermission(endpoint string) (string, bool) {
	perm, ok := s.requiredPermissions[endpoint]
	return perm, ok
}

func invertLeadingSlash(path string) string {
	path, withoutSlash := strings.CutPrefix(path, "/")
	if withoutSlash {
		return path
	}
	return "/" + path
}
