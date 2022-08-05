package routes

import (
	"context"

	"github.com/integration-system/isp-kit/cluster"
)

type Routes struct {
	innerMethods map[string]bool
}

func NewRoutes() *Routes {
	return &Routes{
		innerMethods: make(map[string]bool),
	}
}

func (s *Routes) ReceiveRoutes(ctx context.Context, routes cluster.RoutingConfig) error {
	newInnerMethods := make(map[string]bool)
	for _, backend := range routes {
		for _, v := range backend.Endpoints {
			if v.Inner {
				newInnerMethods[v.Path] = true
			}
		}
	}

	s.innerMethods = newInnerMethods

	return nil
}

func (s *Routes) IsInnerMethod(path string) bool {
	return s.innerMethods[path]
}
