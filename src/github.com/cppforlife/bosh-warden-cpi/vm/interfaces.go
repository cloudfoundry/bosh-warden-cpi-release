package vm

import (
	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
)

type Creator interface {
	Create(string, bwcstem.Stemcell, VMProps, Networks, Environment) (VM, error)
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

type VMProps struct {
	Ports []VMPropsPort
}

type VMPropsPort struct {
	Host      int
	Container int
	Protocol  string
}

type Environment map[string]interface{}

type Ports interface {
	Forward(string, string, []VMPropsPort) error
	RemoveForwarded(string) error
}

type AgentEnvService interface {
	// Fetch will return an error if Update was not called beforehand
	Fetch() (AgentEnv, error)
	Update(AgentEnv) error
}

type AgentEnvServiceFactory interface {
	New(WardenFileService, string) AgentEnvService
}

type GuestBindMounts interface {
	MakeEphemeral() string
	MakePersistent() string
	MountPersistent(diskID string) string
}

type HostBindMounts interface {
	MakeEphemeral(id string) (string, error)
	DeleteEphemeral(id string) error

	MakePersistent(id string) (string, error)
	DeletePersistent(id string) error

	MountPersistent(id, diskID, diskPath string) error
	UnmountPersistent(id, diskID string) error
}

type MetadataService interface {
	Save(WardenFileService, string) error
}

type WardenFileService interface {
	Upload(string, []byte) error
	Download(string) ([]byte, error)
}
