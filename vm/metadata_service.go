package vm

import (
	"encoding/json"
	"fmt"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
)

type metadataService struct {
	agentEnvService  string
	registryOptions  RegistryOptions
	userDataFilePath string
	logger           boshlog.Logger
	logTag           string
}

type MetadataService interface {
	Save(WardenFileService) error
}

func NewMetadataService(
	agentEnvService string,
	registryOptions RegistryOptions,
	logger boshlog.Logger,
) MetadataService {
	return &metadataService{
		agentEnvService:  agentEnvService,
		registryOptions:  registryOptions,
		userDataFilePath: "/var/vcap/bosh/warden-cpi-user-data.json",
		logger:           logger,
		logTag:           "metadataService",
	}
}

type UserDataContentsType struct {
	Registry RegistryType
}

type RegistryType struct {
	Endpoint string
}

func (ms *metadataService) Save(wardenFileService WardenFileService) error {
	var endpoint string

	if ms.agentEnvService == "registry" {
		endpoint = fmt.Sprintf(
			"http://%s:%s@%s:%d",
			ms.registryOptions.Username,
			ms.registryOptions.Password,
			ms.registryOptions.Host,
			ms.registryOptions.Port,
		)
	} else {
		endpoint = "/var/vcap/bosh/warden-cpi-agent-env.json"
	}

	userDataContents := UserDataContentsType{
		Registry: RegistryType{
			Endpoint: endpoint,
		},
	}

	jsonBytes, err := json.Marshal(userDataContents)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling user data")
	}

	ms.logger.Debug(ms.logTag, "Saving user data %#v to %s", userDataContents, ms.userDataFilePath)

	return wardenFileService.Upload(ms.userDataFilePath, jsonBytes)
}
