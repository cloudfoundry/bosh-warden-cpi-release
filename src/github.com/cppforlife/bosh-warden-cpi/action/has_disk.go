package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"

	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
)

type HasDiskMethod struct {
	diskFinder bwcdisk.Finder
}

func NewHasDiskMethod(diskFinder bwcdisk.Finder) HasDiskMethod {
	return HasDiskMethod{diskFinder: diskFinder}
}

func (a HasDiskMethod) HasDisk(cid apiv1.DiskCID) (bool, error) {
	disk, err := a.diskFinder.Find(cid)
	if err != nil {
		return false, bosherr.WrapErrorf(err, "Finding disk '%s'", cid)
	}

	return disk.Exists()
}
