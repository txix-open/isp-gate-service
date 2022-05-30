package routes

import (
	"context"

	"github.com/integration-system/isp-kit/cluster"
)

type Routes struct {
	innerMethods    map[string]bool
	allMethods      map[string]bool
	authUserMethods map[string]bool
}

func NewRoutes() *Routes {
	return &Routes{
		innerMethods:    make(map[string]bool),
		allMethods:      make(map[string]bool),
		authUserMethods: make(map[string]bool),
	}
}

func (s *Routes) ReceiveRoutes(ctx context.Context, routes cluster.RoutingConfig) error {
	newAddressMap := make(map[string]bool)
	newInnerAddressMap := make(map[string]bool)
	newAuthUserAddressMap := make(map[string]bool)
	for _, backend := range routes {
		if backend.Address.IP == "" || backend.Address.Port == "" || len(backend.Endpoints) == 0 {
			continue
		}
		for _, v := range backend.Endpoints {
			newAddressMap[v.Path] = true
			if v.Inner {
				newInnerAddressMap[v.Path] = true
			}
			if v.UserAuthRequired {
				newAuthUserAddressMap[v.Path] = true
			}
		}
	}
	s.allMethods = newAddressMap
	s.innerMethods = newInnerAddressMap
	s.authUserMethods = newAuthUserAddressMap

	return nil
}

func (s *Routes) IsAvailableMethod(path string) bool {
	return s.allMethods[path]
}

func (s *Routes) IsAuthUserMethod(path string) bool {
	return s.authUserMethods[path]
}

func (s *Routes) IsInnerMethod(path string) bool {
	return s.innerMethods[path]
}
