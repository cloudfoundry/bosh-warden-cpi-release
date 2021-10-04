package vm

import (
	bwcdisk "bosh-warden-cpi/disk"
	bwcstem "bosh-warden-cpi/stemcell"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

type Creator interface {
	Create(apiv1.AgentID, bwcstem.Stemcell, VMProps, apiv1.Networks, apiv1.VMEnv) (VM, error)
}

type Finder interface {
	Find(apiv1.VMCID) (VM, bool, error)
}

type VM interface {
	ID() apiv1.VMCID

	Delete() error

	AttachDisk(bwcdisk.Disk) (apiv1.DiskHint, error)
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
