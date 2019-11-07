package routing

import (
	"github.com/integration-system/isp-lib/structure"
)

var (
	InnerAddressMap = make(map[string]bool)
	AddressMap      = make(map[string]bool)
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
	AddressMap = newAddressMap
	InnerAddressMap = newInnerAddressMap
}
