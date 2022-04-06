package vm

import (
	"encoding/json"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type metadataService struct {
	userDataFilePath string
	metadataFilePath string

	logTag string
	logger boshlog.Logger
}

func NewMetadataService(
	logger boshlog.Logger,
) MetadataService {
	return &metadataService{
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

func (ms *metadataService) Save(wardenFileService WardenFileService, instanceID apiv1.VMCID) error {
	userDataContents := UserDataContentsType{
		Registry: RegistryType{
			Endpoint: "/var/vcap/bosh/warden-cpi-agent-env.json",
		},
	}

	jsonBytes, err := json.Marshal(userDataContents)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling user data")
	}

	ms.logger.Debug(ms.logTag, "Saving user data to %s", ms.userDataFilePath)

	err = wardenFileService.Upload(ms.userDataFilePath, jsonBytes)
	if err != nil {
		return bosherr.WrapError(err, "Saving user data")
	}

	metadataContents := MetadataContentsType{
		InstanceID: instanceID.AsString(),
	}

	jsonBytes, err = json.Marshal(metadataContents)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling metadata")
	}

	err = wardenFileService.Upload(ms.metadataFilePath, jsonBytes)
	if err != nil {
		return bosherr.WrapError(err, "Saving metadata")
	}

	return nil
}
