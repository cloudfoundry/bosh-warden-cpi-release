package fakes

import (
	bwcdisk "bosh-warden-cpi/disk"
)

type FakeCreator struct {
	CreateSize int
	CreateDisk bwcdisk.Disk
	CreateErr  error
}

func (c *FakeCreator) Create(size int) (bwcdisk.Disk, error) {
	c.CreateSize = size
	return c.CreateDisk, c.CreateErr
}
