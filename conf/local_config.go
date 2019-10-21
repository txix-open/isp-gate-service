package conf

import "github.com/integration-system/isp-lib/structure"

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
		PathPrefix   string
		Protocol     string
		TargetModule string
	}
)
