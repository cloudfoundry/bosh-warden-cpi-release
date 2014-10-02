package action

import (
	bosherr "github.com/cloudfoundry/bosh-agent/errors"

	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type DeleteVM struct {
	vmFinder bwcvm.Finder
}

func NewDeleteVM(vmFinder bwcvm.Finder) DeleteVM {
	return DeleteVM{vmFinder: vmFinder}
}

func (a DeleteVM) Run(vmCID VMCID) (interface{}, error) {
	vm, found, err := a.vmFinder.Find(string(vmCID))
	if err != nil {
		return nil, bosherr.WrapError(err, "Finding vm '%s'", vmCID)
	}

	if found {
		err := vm.Delete()
		if err != nil {
			return nil, bosherr.WrapError(err, "Deleting vm '%s'", vmCID)
		}
	}

	return nil, nil
}
