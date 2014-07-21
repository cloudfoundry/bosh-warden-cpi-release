package fakes

import (
	bwcdisk "bosh-warden-cpi/disk"
)

type FakeFinder struct {
	FindID    string
	FindDisk  bwcdisk.Disk
	FindFound bool
	FindErr   error
}

func (f *FakeFinder) Find(id string) (bwcdisk.Disk, bool, error) {
	f.FindID = id
	return f.FindDisk, f.FindFound, f.FindErr
}
