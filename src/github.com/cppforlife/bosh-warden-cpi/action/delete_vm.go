package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type DeleteVM struct {
	vmFinder       bwcvm.Finder
	hostBindMounts bwcvm.HostBindMounts
}

func NewDeleteVM(vmFinder bwcvm.Finder, hostBindMounts bwcvm.HostBindMounts) DeleteVM {
	return DeleteVM{vmFinder: vmFinder, hostBindMounts: hostBindMounts}
}

func (a DeleteVM) Run(vmCID VMCID) (interface{}, error) {
	vm, _, err := a.vmFinder.Find(string(vmCID))
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Finding vm '%s'", vmCID)
	}

	err = vm.Delete()
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Deleting vm '%s'", vmCID)
	}

	return nil, nil
}
