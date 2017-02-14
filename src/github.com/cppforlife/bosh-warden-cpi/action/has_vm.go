package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"

	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type HasVMMethod struct {
	vmFinder bwcvm.Finder
}

func NewHasVMMethod(vmFinder bwcvm.Finder) HasVMMethod {
	return HasVMMethod{vmFinder: vmFinder}
}

func (a HasVMMethod) HasVM(cid apiv1.VMCID) (bool, error) {
	_, found, err := a.vmFinder.Find(cid)
	if err != nil {
		return false, bosherr.WrapErrorf(err, "Finding VM '%s'", cid)
	}

	return found, nil
}
