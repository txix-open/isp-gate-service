package conf

import (
	"github.com/integration-system/isp-lib/v2/config"
	"github.com/integration-system/isp-lib/v2/structure"
)

type (
	Configuration struct {
		config.CommonLocalConfig
		InstanceUuid     string                         `valid:"required~Required"`
		HttpOuterAddress structure.AddressConfiguration `valid:"required~Required" json:"httpOuterAddress"`
		HttpInnerAddress structure.AddressConfiguration `valid:"required~Required" json:"httpInnerAddress"`
		Locations        []Location
	}

	Location struct {
		SkipAuth       bool
		SkipExistCheck bool
		PathPrefix     string `valid:"required~Required"`
		Protocol       string `valid:"required~Required"`
		TargetModule   string `valid:"required~Required"`
	}
)

func GetLocationsByTargetModule(locations []Location) map[string][]Location {
	requiredModules := make(map[string][]Location)

	for _, loc := range locations {
		requiredModules[loc.TargetModule] = append(requiredModules[loc.TargetModule], loc)
	}

	return requiredModules
}
