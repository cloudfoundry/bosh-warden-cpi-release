package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
)

type CreateDisk struct {
	diskCreator bwcdisk.Creator
}

type DiskCloudProperties map[string]interface{}

func NewCreateDisk(diskCreator bwcdisk.Creator) CreateDisk {
	return CreateDisk{diskCreator: diskCreator}
}

func (a CreateDisk) Run(size int, _ DiskCloudProperties, _ VMCID) (DiskCID, error) {
	disk, err := a.diskCreator.Create(size)
	if err != nil {
		return "", bosherr.WrapError(err, "Creating disk of size '%d'", size)
	}

	return DiskCID(disk.ID()), nil
}
