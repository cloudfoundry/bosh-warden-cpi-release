package fakes

import (
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type FakeMetadataService struct {
	Saved   bool
	SaveErr error
}

func NewFakeMetadataService() *FakeMetadataService {
	return &FakeMetadataService{}
}

func (ms *FakeMetadataService) Save(wardenFileService bwcvm.WardenFileService) error {
	ms.Saved = true

	return ms.SaveErr
}
