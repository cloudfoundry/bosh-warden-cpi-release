package vm

import (
	bwcdisk "bosh-warden-cpi/disk"
	bwcstem "bosh-warden-cpi/stemcell"
)

type Creator interface {
	// Create takes an agent id and creates a VM with provided configuration
	Create(string, bwcstem.Stemcell, Networks, Environment) (VM, error)
}

type Finder interface {
	Find(string) (VM, bool, error)
}

type VM interface {
	ID() string

	Delete() error

	AttachDisk(bwcdisk.Disk) error
	DetachDisk(bwcdisk.Disk) error
}

type Environment map[string]interface{}
