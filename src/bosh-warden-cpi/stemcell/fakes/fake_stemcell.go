package fakes

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

type FakeStemcell struct {
	id  apiv1.StemcellCID
	uri string

	DeleteCalled bool
	DeleteErr    error
}

func NewFakeStemcell(id apiv1.StemcellCID) *FakeStemcell {
	return &FakeStemcell{id: id}
}

func NewFakeStemcellWithPath(id apiv1.StemcellCID, uri string) *FakeStemcell {
	return &FakeStemcell{id: id, uri: uri}
}

func (s FakeStemcell) ID() apiv1.StemcellCID { return s.id }

func (s FakeStemcell) URI() string { return s.uri }

func (s *FakeStemcell) Delete() error {
	s.DeleteCalled = true
	return s.DeleteErr
}
