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
		RouterConfig         RouterConfig                   `valid:"required~Required" json:"routerConfig"`
	}

	RouterConfig struct {
		RouterModuleName string `valid:"required~Required"`
		PathPrefix       string `valid:"required~Required"`
		SkipAuth         bool
		SkipExistCheck   bool
	}
)
