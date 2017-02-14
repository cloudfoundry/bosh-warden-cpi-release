package vm

import (
	gobytes "bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
)

type registryAgentEnvService struct {
	endpoint string
	logger   boshlog.Logger
	logTag   string
}

type registryResp struct {
	Settings string
}

func NewRegistryAgentEnvService(
	registryOptions RegistryOptions,
	instanceID apiv1.VMCID,
	logger boshlog.Logger,
) AgentEnvService {
	endpoint := fmt.Sprintf(
		"http://%s:%s@%s:%d/instances/%s/settings",
		registryOptions.Username,
		registryOptions.Password,
		registryOptions.Host,
		registryOptions.Port,
		instanceID.AsString(),
	)
	return registryAgentEnvService{
		endpoint: endpoint,
		logger:   logger,
		logTag:   "vm.registryAgentEnvService",
	}
}

func (s registryAgentEnvService) Fetch() (apiv1.AgentEnv, error) {
	s.logger.Debug(s.logTag, "Fetching agent env from registry endpoint %s", s.endpoint)

	httpClient := http.Client{}
	httpResponse, err := httpClient.Get(s.endpoint)
	if err != nil {
		return nil, bosherr.WrapError(err, "Fetching agent env from registry")
	}

	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, bosherr.Errorf("Received non-200 status code when contacting registry: '%d'", httpResponse.StatusCode)
	}

	httpBody, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Reading response from registry endpoint '%s'", s.endpoint)
	}

	var resp registryResp

	err = json.Unmarshal(httpBody, &resp)
	if err != nil {
		return nil, bosherr.WrapError(err, "Unmarshalling registry response")
	}

	agentEnv, err := apiv1.AgentEnvFactory{}.FromBytes([]byte(resp.Settings))
	if err != nil {
		return nil, bosherr.WrapError(err, "Unmarshalling agent env from registry")
	}

	return agentEnv, nil
}

func (s registryAgentEnvService) Update(agentEnv apiv1.AgentEnv) error {
	bytes, err := agentEnv.AsBytes()
	if err != nil {
		return bosherr.WrapError(err, "Marshalling agent env")
	}

	request, err := http.NewRequest("PUT", s.endpoint, gobytes.NewReader(bytes))
	if err != nil {
		return bosherr.WrapErrorf(err, "Creating PUT request to update registry at '%s'", s.endpoint)
	}

	httpClient := http.Client{}
	httpResponse, err := httpClient.Do(request)
	if err != nil {
		return bosherr.WrapErrorf(err, "Updating registry endpoint '%s'", s.endpoint)
	}

	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK && httpResponse.StatusCode != http.StatusCreated {
		return bosherr.Errorf("Received non-2xx status code when contacting registry: '%d'", httpResponse.StatusCode)
	}

	return nil
}
