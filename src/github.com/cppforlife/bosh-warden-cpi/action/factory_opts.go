package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	"github.com/cppforlife/bosh-cpi-go/apiv1"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type FactoryOpts struct {
	StemcellsDir string
	DisksDir     string

	HostEphemeralBindMountsDir  string // e.g. /var/vcap/store/ephemeral_disks
	HostPersistentBindMountsDir string // e.g. /var/vcap/store/persistent_disks

	GuestEphemeralBindMountPath  string // e.g. /var/vcap/data
	GuestPersistentBindMountsDir string // e.g. /warden-cpi-dev

	Agent apiv1.AgentOptions

	AgentEnvService string
	Registry        bwcvm.RegistryOptions
}

func (o FactoryOpts) Validate() error {
	if o.StemcellsDir == "" {
		return bosherr.Error("Must provide non-empty StemcellsDir")
	}

	if o.DisksDir == "" {
		return bosherr.Error("Must provide non-empty DisksDir")
	}

	if o.HostEphemeralBindMountsDir == "" {
		return bosherr.Error("Must provide non-empty HostEphemeralBindMountsDir")
	}

	if o.HostPersistentBindMountsDir == "" {
		return bosherr.Error("Must provide non-empty HostPersistentBindMountsDir")
	}

	if o.GuestEphemeralBindMountPath == "" {
		return bosherr.Error("Must provide non-empty GuestEphemeralBindMountPath")
	}

	if o.GuestPersistentBindMountsDir == "" {
		return bosherr.Error("Must provide non-empty GuestPersistentBindMountsDir")
	}

	err := o.Agent.Validate()
	if err != nil {
		return bosherr.WrapError(err, "Validating Agent configuration")
	}

	return nil
}
