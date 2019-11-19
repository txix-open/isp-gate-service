package routing

import (
	"github.com/integration-system/isp-lib/structure"
)

var (
	InnerMethods = make(map[string]bool)
	AllMethods   = make(map[string]bool)
)

func InitRoutes(configs structure.RoutingConfig) {
	newAddressMap := make(map[string]bool)
	newInnerAddressMap := make(map[string]bool)
	for _, backend := range configs {
		if backend.Address.IP == "" || backend.Address.Port == "" || len(backend.Endpoints) == 0 {
			continue
		}
		for _, v := range backend.Endpoints {
			newAddressMap[v.Path] = true
			if v.Inner {
				newInnerAddressMap[v.Path] = true
			}
		}
	}
	AllMethods = newAddressMap
	InnerMethods = newInnerAddressMap
}