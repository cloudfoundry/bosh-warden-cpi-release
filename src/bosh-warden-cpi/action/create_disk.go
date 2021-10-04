package action

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	bwcdisk "bosh-warden-cpi/disk"
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
