package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type AttachDisk struct {
	vmFinder   bwcvm.Finder
	diskFinder bwcdisk.Finder
}

func NewAttachDisk(vmFinder bwcvm.Finder, diskFinder bwcdisk.Finder) AttachDisk {
	return AttachDisk{vmFinder, diskFinder}
}

func (a AttachDisk) Run(vmCID VMCID, diskCID DiskCID) (interface{}, error) {
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

	err = vm.AttachDisk(disk)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Attaching disk '%s' to VM '%s'", diskCID, vmCID)
	}

	return nil, nil
}
