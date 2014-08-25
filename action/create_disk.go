package action

import (
	bosherr "bosh/errors"

	bwcdisk "bosh-warden-cpi/disk"
)

type CreateDisk struct {
	diskCreator bwcdisk.Creator
}

func NewCreateDisk(diskCreator bwcdisk.Creator) CreateDisk {
	return CreateDisk{diskCreator: diskCreator}
}

func (a CreateDisk) Run(size int, _ map[string]string, _ VMCID) (DiskCID, error) {
	disk, err := a.diskCreator.Create(size)
	if err != nil {
		return "", bosherr.WrapError(err, "Creating disk of size '%d'", size)
	}

	return DiskCID(disk.ID()), nil
}
