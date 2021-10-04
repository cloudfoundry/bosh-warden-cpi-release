package fakes

import (
	bwcdisk "bosh-warden-cpi/disk"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

type FakeFactory struct {
	CreateSize int
	CreateDisk bwcdisk.Disk
	CreateErr  error

	FindID   apiv1.DiskCID
	FindDisk bwcdisk.Disk
	FindErr  error
}

func (f *FakeFactory) Create(size int) (bwcdisk.Disk, error) {
	f.CreateSize = size
	return f.CreateDisk, f.CreateErr
}

func (f *FakeFactory) Find(id apiv1.DiskCID) (bwcdisk.Disk, error) {
	f.FindID = id
	return f.FindDisk, f.FindErr
}
