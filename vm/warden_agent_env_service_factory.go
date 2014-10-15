package vm

import (
	wrdn "github.com/cloudfoundry-incubator/garden/warden"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
)

type WardenAgentEnvServiceFactory struct {
	logger          boshlog.Logger
	agentEnvService string
	registryOptions RegistryOptions
}

func NewWardenAgentEnvServiceFactory(
	logger boshlog.Logger,
	agentEnvService string,
	registryOptions RegistryOptions,
) WardenAgentEnvServiceFactory {
	return WardenAgentEnvServiceFactory{
		logger:          logger,
		agentEnvService: agentEnvService,
		registryOptions: registryOptions,
	}
}

func (f WardenAgentEnvServiceFactory) New(container wrdn.Container, instanceID string) AgentEnvService {
	if f.agentEnvService == "registry" {
		return NewRegistryAgentEnvService(f.registryOptions, instanceID, f.logger)
	}
	return NewFSAgentEnvService(container, f.logger)
}
