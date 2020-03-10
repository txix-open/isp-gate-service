package conf

import (
	"github.com/integration-system/isp-lib/v2/structure"
)

type (
	Configuration struct {
		InstanceUuid         string                         `valid:"required~Required"`
		ModuleName           string                         `valid:"required~Required"`
		ConfigServiceAddress structure.AddressConfiguration `valid:"required~Required"`
		HttpOuterAddress     structure.AddressConfiguration `valid:"required~Required" json:"httpOuterAddress"`
		HttpInnerAddress     structure.AddressConfiguration `valid:"required~Required" json:"httpInnerAddress"`
		Locations            []Location
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

	for _, location := range locations {
		if location.TargetModule == "" {
			continue
		}
		if loc, ok := requiredModules[location.TargetModule]; ok {
			requiredModules[location.TargetModule] = append(loc, location)
		} else {
			requiredModules[location.TargetModule] = []Location{location}
		}
	}

	return requiredModules
}
