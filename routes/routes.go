package routes

import (
	"context"
	"strings"

	"github.com/txix-open/isp-kit/cluster"
)

type Routes struct {
	innerEndpoints      map[string]bool
	requiredPermissions map[string]string
}

func NewRoutes() *Routes {
	return &Routes{
		innerEndpoints:      make(map[string]bool),
		requiredPermissions: make(map[string]string),
	}
}

func (s *Routes) ReceiveRoutes(ctx context.Context, routes cluster.RoutingConfig) error {
	newInnerMethods := make(map[string]bool)
	newRequiredPermissions := make(map[string]string)
	for _, backend := range routes {
		for _, v := range backend.Endpoints {
			if v.Inner {
				newInnerMethods[v.Path] = true
			}
			perm, ok := cluster.GetRequiredAdminPermission(v)
			if ok {
				newRequiredPermissions[v.Path] = perm
			}
		}
	}

	s.innerEndpoints = newInnerMethods
	s.requiredPermissions = newRequiredPermissions

	return nil
}

func (s *Routes) IsInnerEndpoint(path string) bool {
	return s.innerEndpoints[path] || s.innerEndpoints[invertLeadingSlash(path)]
}

func (s *Routes) RequiredAdminPermission(path string) (string, bool) {
	perm, ok := s.requiredPermissions[path]
	if !ok {
		perm, ok = s.requiredPermissions[invertLeadingSlash(path)]
	}
	return perm, ok
}

func invertLeadingSlash(path string) string {
	path, withoutSlash := strings.CutPrefix(path, "/")
	if withoutSlash {
		return path
	}
	return "/" + path
}
