package fakes

import (
	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
)

type FakeFactory struct {
	CreateSize int
	CreateDisk bwcdisk.Disk
	CreateErr  error

	FindID   string
	FindDisk bwcdisk.Disk
	FindErr  error
}

func (f *FakeFactory) Create(size int) (bwcdisk.Disk, error) {
	f.CreateSize = size
	return f.CreateDisk, f.CreateErr
}

func (f *FakeFactory) Find(id string) (bwcdisk.Disk, error) {
	f.FindID = id
	return f.FindDisk, f.FindErr
}
