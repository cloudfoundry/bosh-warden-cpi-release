package vm

import (
	"encoding/json"
	"fmt"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type metadataService struct {
	agentEnvService  string
	registryOptions  RegistryOptions
	userDataFilePath string
	metadataFilePath string

	logTag string
	logger boshlog.Logger
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
		metadataFilePath: "/var/vcap/bosh/warden-cpi-metadata.json",

		logTag: "vm.metadataService",
		logger: logger,
	}
}

type RegistryType struct {
	Endpoint string
}

type UserDataContentsType struct {
	Registry RegistryType
}

type MetadataContentsType struct {
	InstanceID string `json:"instance-id"`
}

func (ms *metadataService) Save(wardenFileService WardenFileService, instanceID string) error {
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

	err = wardenFileService.Upload(ms.userDataFilePath, jsonBytes)
	if err != nil {
		return bosherr.WrapError(err, "Saving user data")
	}

	metadataContents := MetadataContentsType{
		InstanceID: instanceID,
	}

	jsonBytes, err = json.Marshal(metadataContents)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling metadata")
	}

	ms.logger.Debug(ms.logTag, "Saving metadata %#v to %s", metadataContents, ms.metadataFilePath)

	err = wardenFileService.Upload(ms.metadataFilePath, jsonBytes)
	if err != nil {
		return bosherr.WrapError(err, "Saving metadata")
	}

	return nil
}
