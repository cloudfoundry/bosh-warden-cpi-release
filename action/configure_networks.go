package action

import (
	bwcapi "github.com/cppforlife/bosh-warden-cpi/api"
)

type ConfigureNetworks struct{}

func NewConfigureNetworks() ConfigureNetworks {
	return ConfigureNetworks{}
}

func (a ConfigureNetworks) Run(vmCID VMCID, networks Networks) (interface{}, error) {
	return nil, bwcapi.NotSupportedError{}
}
