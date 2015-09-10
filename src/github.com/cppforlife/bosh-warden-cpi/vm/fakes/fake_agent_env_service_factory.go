package fakes

import (
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type FakeAgentEnvServiceFactory struct {
	NewWardenFileService bwcvm.WardenFileService
	NewInstanceID        string
	NewAgentEnvService   *FakeAgentEnvService
}

func (f *FakeAgentEnvServiceFactory) New(
	wardenFileService bwcvm.WardenFileService,
	instanceID string,
) bwcvm.AgentEnvService {
	f.NewWardenFileService = wardenFileService
	f.NewInstanceID = instanceID

	if f.NewAgentEnvService == nil {
		// Always return non-nil service for convenience
		return &FakeAgentEnvService{}
	}

	return f.NewAgentEnvService
}
