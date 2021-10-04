package fakes

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

type FakeDisk struct {
	id   apiv1.DiskCID
	path string

	DeleteCalled bool
	DeleteErr    error
}

func NewFakeDisk(id apiv1.DiskCID) *FakeDisk {
	return &FakeDisk{id: id}
}

func NewFakeDiskWithPath(id apiv1.DiskCID, path string) *FakeDisk {
	return &FakeDisk{id: id, path: path}
}

func (s FakeDisk) ID() apiv1.DiskCID { return s.id }

func (s FakeDisk) Path() string { return s.path }

func (s *FakeDisk) Exists() (bool, error) { return false, nil }

func (s *FakeDisk) Delete() error {
	s.DeleteCalled = true
	return s.DeleteErr
}
