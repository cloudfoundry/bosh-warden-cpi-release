package fakes

import (
	bwcstem "bosh-warden-cpi/stemcell"
)

type FakeFinder struct {
	FindID       string
	FindStemcell bwcstem.Stemcell
	FindFound    bool
	FindErr      error
}

func (f *FakeFinder) Find(id string) (bwcstem.Stemcell, bool, error) {
	f.FindID = id
	return f.FindStemcell, f.FindFound, f.FindErr
}
