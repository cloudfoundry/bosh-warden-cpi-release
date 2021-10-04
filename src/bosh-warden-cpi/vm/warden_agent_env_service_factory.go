package vm

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type WardenAgentEnvServiceFactory struct {
	agentEnvService string
	registryOptions RegistryOptions
	logger          boshlog.Logger
}

func NewWardenAgentEnvServiceFactory(
	agentEnvService string,
	registryOptions RegistryOptions,
	logger boshlog.Logger,
) WardenAgentEnvServiceFactory {
	return WardenAgentEnvServiceFactory{
		logger:          logger,
		agentEnvService: agentEnvService,
		registryOptions: registryOptions,
	}
}

func (f WardenAgentEnvServiceFactory) New(
	wardenFileService WardenFileService,
	instanceID apiv1.VMCID,
) AgentEnvService {
	if f.agentEnvService == "registry" {
		return NewRegistryAgentEnvService(f.registryOptions, instanceID, f.logger)
	}
	return NewFSAgentEnvService(wardenFileService, f.logger)
}
