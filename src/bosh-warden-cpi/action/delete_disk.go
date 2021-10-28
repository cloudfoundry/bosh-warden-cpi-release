package action

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	bwcdisk "bosh-warden-cpi/disk"
)

type DeleteDiskMethod struct {
	diskFinder bwcdisk.Finder
}

func NewDeleteDiskMethod(diskFinder bwcdisk.Finder) DeleteDiskMethod {
	return DeleteDiskMethod{diskFinder: diskFinder}
}

func (a DeleteDiskMethod) DeleteDisk(cid apiv1.DiskCID) error {
	disk, err := a.diskFinder.Find(cid)
	if err != nil {
		return bosherr.WrapErrorf(err, "Finding disk '%s'", cid)
	}

	err = disk.Delete()
	if err != nil {
		return bosherr.WrapErrorf(err, "Deleting disk '%s'", cid)
	}

	return nil
}
