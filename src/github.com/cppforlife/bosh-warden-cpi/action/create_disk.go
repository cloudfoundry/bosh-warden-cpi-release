package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"

	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
)

type CreateDiskMethod struct {
	diskCreator bwcdisk.Creator
}

func NewCreateDiskMethod(diskCreator bwcdisk.Creator) CreateDiskMethod {
	return CreateDiskMethod{diskCreator: diskCreator}
}

func (a CreateDiskMethod) CreateDisk(size int, _ apiv1.DiskCloudProps, _ *apiv1.VMCID) (apiv1.DiskCID, error) {
	disk, err := a.diskCreator.Create(size)
	if err != nil {
		return apiv1.DiskCID{}, bosherr.WrapErrorf(err, "Creating disk of size '%d'", size)
	}

	return disk.ID(), nil
}
