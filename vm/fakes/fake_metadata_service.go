package fakes

import (
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type FakeMetadataService struct {
	Saved          bool
	SaveInstanceID string
	SaveErr        error
}

func NewFakeMetadataService() *FakeMetadataService {
	return &FakeMetadataService{}
}

func (ms *FakeMetadataService) Save(wardenFileService bwcvm.WardenFileService, instanceID string) error {
	ms.Saved = true
	ms.SaveInstanceID = instanceID

	return ms.SaveErr
}
