package vm

import (
	wrdn "github.com/cloudfoundry-incubator/garden/warden"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
)

type WardenAgentEnvServiceFactory struct {
	logger boshlog.Logger
}

func NewWardenAgentEnvServiceFactory(logger boshlog.Logger) WardenAgentEnvServiceFactory {
	return WardenAgentEnvServiceFactory{logger: logger}
}

func (f WardenAgentEnvServiceFactory) New(container wrdn.Container) AgentEnvService {
	return NewFSAgentEnvService(container, f.logger)
}
