package fakes

import (
	"github.com/cppforlife/bosh-cpi-go/apiv1"

	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type FakeAgentEnvServiceFactory struct {
	NewWardenFileService bwcvm.WardenFileService
	NewInstanceID        apiv1.VMCID
	NewAgentEnvService   *FakeAgentEnvService
}

func (f *FakeAgentEnvServiceFactory) New(
	wardenFileService bwcvm.WardenFileService,
	instanceID apiv1.VMCID,
) bwcvm.AgentEnvService {
	f.NewWardenFileService = wardenFileService
	f.NewInstanceID = instanceID

	if f.NewAgentEnvService == nil {
		// Always return non-nil service for convenience
		return &FakeAgentEnvService{}
	}

	return f.NewAgentEnvService
}
