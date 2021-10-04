package vm

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type fsAgentEnvService struct {
	wardenFileService WardenFileService
	settingsPath      string

	logTag string
	logger boshlog.Logger
}

func NewFSAgentEnvService(
	wardenFileService WardenFileService,
	logger boshlog.Logger,
) AgentEnvService {
	return fsAgentEnvService{
		wardenFileService: wardenFileService,
		settingsPath:      "/var/vcap/bosh/warden-cpi-agent-env.json",

		logTag: "vm.FSAgentEnvService",
		logger: logger,
	}
}

func (s fsAgentEnvService) Fetch() (apiv1.AgentEnv, error) {
	bytes, err := s.wardenFileService.Download(s.settingsPath)
	if err != nil {
		return nil, bosherr.WrapError(err, "Downloading agent env from container")
	}

	return apiv1.AgentEnvFactory{}.FromBytes(bytes)
}

func (s fsAgentEnvService) Update(agentEnv apiv1.AgentEnv) error {
	bytes, err := agentEnv.AsBytes()
	if err != nil {
		return bosherr.WrapError(err, "Marshalling agent env")
	}

	return s.wardenFileService.Upload(s.settingsPath, bytes)
}
