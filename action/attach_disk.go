package action

import (
	bosherr "bosh/errors"

	bwcdisk "bosh-warden-cpi/disk"
	bwcvm "bosh-warden-cpi/vm"
)

type AttachDisk struct {
	vmFinder   bwcvm.Finder
	diskFinder bwcdisk.Finder
}

func NewAttachDisk(vmFinder bwcvm.Finder, diskFinder bwcdisk.Finder) AttachDisk {
	return AttachDisk{
		vmFinder:   vmFinder,
		diskFinder: diskFinder,
	}
}

func (a AttachDisk) Run(vmCID VMCID, diskCID DiskCID) (interface{}, error) {
	vm, found, err := a.vmFinder.Find(string(vmCID))
	if err != nil {
		return nil, bosherr.WrapError(err, "Finding VM '%s'", vmCID)
	}

	if !found {
		return nil, bosherr.New("Expected to find VM '%s'", vmCID)
	}

	disk, found, err := a.diskFinder.Find(string(diskCID))
	if err != nil {
		return nil, bosherr.WrapError(err, "Finding disk '%s'", diskCID)
	}

	if !found {
		return nil, bosherr.New("Expected to find disk '%s'", diskCID)
	}

	err = vm.AttachDisk(disk)
	if err != nil {
		return nil, bosherr.WrapError(err, "Attaching disk '%s' to VM '%s'", diskCID, vmCID)
	}

	return nil, nil
}
