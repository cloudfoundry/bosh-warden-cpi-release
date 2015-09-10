package fakes

import (
	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
)

type FakeImporter struct {
	ImportFromPathImagePath string
	ImportFromPathStemcell  bwcstem.Stemcell
	ImportFromPathErr       error
}

func (c *FakeImporter) ImportFromPath(imagePath string) (bwcstem.Stemcell, error) {
	c.ImportFromPathImagePath = imagePath
	return c.ImportFromPathStemcell, c.ImportFromPathErr
}
