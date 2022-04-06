package vm

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type WardenAgentEnvServiceFactory struct {
	logger boshlog.Logger
}

func NewWardenAgentEnvServiceFactory(
	logger boshlog.Logger,
) WardenAgentEnvServiceFactory {
	return WardenAgentEnvServiceFactory{
		logger: logger,
	}
}

func (f WardenAgentEnvServiceFactory) New(
	wardenFileService WardenFileService,
	instanceID apiv1.VMCID,
) AgentEnvService {
	return NewFSAgentEnvService(wardenFileService, f.logger)
}
