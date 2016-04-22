package action

import "github.com/cppforlife/bosh-warden-cpi/vm"

type VMCloudProperties struct {
	LaunchUpstart bool `json:"launch_upstart"`
}

func (vcp VMCloudProperties) AsVMCloudProperties() vm.CloudProperties {
	return vm.CloudProperties{
		LaunchUpstart: vcp.LaunchUpstart,
	}
}
