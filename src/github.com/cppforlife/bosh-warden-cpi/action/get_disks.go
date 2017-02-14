package action

import (
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
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
