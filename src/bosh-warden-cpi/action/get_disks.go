package action

import (
	bwcvm "bosh-warden-cpi/vm"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

type GetDisksMethod struct {
	vmFinder bwcvm.Finder
}

func NewGetDisksMethod(vmFinder bwcvm.Finder) GetDisksMethod {
	return GetDisksMethod{vmFinder}
}

func (a GetDisksMethod) GetDisks(cid apiv1.VMCID) ([]apiv1.DiskCID, error) {
	// todo implement
	return nil, nil
}
