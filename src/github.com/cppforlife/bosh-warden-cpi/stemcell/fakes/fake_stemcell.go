package fakes

import (
	"github.com/cppforlife/bosh-cpi-go/apiv1"
)

type FakeStemcell struct {
	id      apiv1.StemcellCID
	dirPath string

	DeleteCalled bool
	DeleteErr    error
}

func NewFakeStemcell(id apiv1.StemcellCID) *FakeStemcell {
	return &FakeStemcell{id: id}
}

func NewFakeStemcellWithPath(id apiv1.StemcellCID, dirPath string) *FakeStemcell {
	return &FakeStemcell{id: id, dirPath: dirPath}
}

func (s FakeStemcell) ID() apiv1.StemcellCID { return s.id }

func (s FakeStemcell) DirPath() string { return s.dirPath }

func (s *FakeStemcell) Delete() error {
	s.DeleteCalled = true
	return s.DeleteErr
}
