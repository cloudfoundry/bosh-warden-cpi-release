package fakes

import (
	bwcvm "bosh-warden-cpi/vm"
	wrdn "github.com/cloudfoundry-incubator/garden/warden"
)

type FakeAgentEnvServiceFactory struct {
	NewContainer       wrdn.Container
	NewAgentEnvService *FakeAgentEnvService
}

func (f *FakeAgentEnvServiceFactory) New(container wrdn.Container) bwcvm.AgentEnvService {
	f.NewContainer = container

	if f.NewAgentEnvService == nil {
		// Always return non-nil service for convenience
		return &FakeAgentEnvService{}
	}

	return f.NewAgentEnvService
}
