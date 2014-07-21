package action

import (
	bosherr "bosh/errors"

	bwcdisk "bosh-warden-cpi/disk"
)

type DeleteDisk struct {
	diskFinder bwcdisk.Finder
}

func NewDeleteDisk(diskFinder bwcdisk.Finder) DeleteDisk {
	return DeleteDisk{diskFinder: diskFinder}
}

func (a DeleteDisk) Run(diskCID DiskCID) (interface{}, error) {
	disk, found, err := a.diskFinder.Find(string(diskCID))
	if err != nil {
		return nil, bosherr.WrapError(err, "Finding disk '%s'", diskCID)
	}

	if found {
		err := disk.Delete()
		if err != nil {
			return nil, bosherr.WrapError(err, "Deleting disk '%s'", diskCID)
		}
	}

	return nil, nil
}
