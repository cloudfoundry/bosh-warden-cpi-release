package fakes

import (
	wrdn "github.com/cloudfoundry-incubator/garden/warden"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type FakeAgentEnvServiceFactory struct {
	NewContainer       wrdn.Container
	NewInstanceID      string
	NewAgentEnvService *FakeAgentEnvService
}

func (f *FakeAgentEnvServiceFactory) New(container wrdn.Container, instanceID string) bwcvm.AgentEnvService {
	f.NewContainer = container
	f.NewInstanceID = instanceID

	if f.NewAgentEnvService == nil {
		// Always return non-nil service for convenience
		return &FakeAgentEnvService{}
	}

	return f.NewAgentEnvService
}
