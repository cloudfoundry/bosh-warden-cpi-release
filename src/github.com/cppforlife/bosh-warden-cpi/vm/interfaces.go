package vm

import (
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
)

//go:generate counterfeiter -o fakes/fake_creator.go . Creator
type Creator interface {
	Create(apiv1.AgentID, bwcstem.Stemcell, VMProps, apiv1.Networks, apiv1.VMEnv) (VM, error)
}

//go:generate counterfeiter -o fakes/fake_finder.go . Finder
type Finder interface {
	Find(apiv1.VMCID) (VM, bool, error)
}

//go:generate counterfeiter -o fakes/fake_vm.go . VM
type VM interface {
	ID() apiv1.VMCID

	Delete() error

	AttachDisk(bwcdisk.Disk) (string, error)
	DetachDisk(bwcdisk.Disk) error
}

type VMProps struct {
	PortMappings []PortMapping
}

type Ports interface {
	Forward(apiv1.VMCID, string, []PortMapping) error
	RemoveForwarded(apiv1.VMCID) error
}

type AgentEnvService interface {
	// Fetch will return an error if Update was not called beforehand
	Fetch() (apiv1.AgentEnv, error)
	Update(apiv1.AgentEnv) error
}

type AgentEnvServiceFactory interface {
	New(WardenFileService, apiv1.VMCID) AgentEnvService
}

type GuestBindMounts interface {
	MakeEphemeral() string
	MakePersistent() string
	MountPersistent(apiv1.DiskCID) string
}

type HostBindMounts interface {
	MakeEphemeral(apiv1.VMCID) (string, error)
	DeleteEphemeral(apiv1.VMCID) error

	MakePersistent(apiv1.VMCID) (string, error)
	DeletePersistent(apiv1.VMCID) error

	MountPersistent(apiv1.VMCID, apiv1.DiskCID, string) error
	UnmountPersistent(apiv1.VMCID, apiv1.DiskCID) error
}

type MetadataService interface {
	Save(WardenFileService, apiv1.VMCID) error
}

type WardenFileService interface {
	Upload(string, []byte) error
	Download(string) ([]byte, error)
}
