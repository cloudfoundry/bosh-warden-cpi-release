package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"

	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type AttachDiskMethod struct {
	vmFinder   bwcvm.Finder
	diskFinder bwcdisk.Finder
	versions   apiv1.ApiVersions
}

func NewAttachDiskMethod(vmFinder bwcvm.Finder, diskFinder bwcdisk.Finder, versions apiv1.ApiVersions) AttachDiskMethod {
	return AttachDiskMethod{vmFinder, diskFinder, versions}
}

func (a AttachDiskMethod) AttachDisk(vmCID apiv1.VMCID, diskCID apiv1.DiskCID) (interface{}, error) {
	vm, found, err := a.vmFinder.Find(vmCID)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Finding VM '%s'", vmCID)
	}

	if !found {
		return nil, bosherr.Errorf("Expected to find VM '%s'", vmCID)
	}

	disk, err := a.diskFinder.Find(diskCID)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Finding disk '%s'", diskCID)
	}

	diskHint, err := vm.AttachDisk(disk)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Attaching disk '%s' to VM '%s'", diskCID, vmCID)
	}

	if a.versions.Contract == 2 {
		return diskHint, nil
	} else {
		return nil, nil
	}
}
