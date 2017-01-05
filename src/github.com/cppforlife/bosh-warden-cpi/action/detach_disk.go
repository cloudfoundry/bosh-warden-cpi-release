package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type DetachDisk struct {
	vmFinder   bwcvm.Finder
	diskFinder bwcdisk.Finder
}

func NewDetachDisk(vmFinder bwcvm.Finder, diskFinder bwcdisk.Finder) DetachDisk {
	return DetachDisk{vmFinder, diskFinder}
}

func (a DetachDisk) Run(vmCID VMCID, diskCID DiskCID) (interface{}, error) {
	vm, found, err := a.vmFinder.Find(string(vmCID))
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Finding VM '%s'", vmCID)
	}

	if !found {
		return nil, bosherr.Errorf("Expected to find VM '%s'", vmCID)
	}

	disk, err := a.diskFinder.Find(string(diskCID))
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Finding disk '%s'", diskCID)
	}

	err = vm.DetachDisk(disk)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Detaching disk '%s' to VM '%s'", diskCID, vmCID)
	}

	return nil, nil
}
