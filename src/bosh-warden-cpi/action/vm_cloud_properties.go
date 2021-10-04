package action

import (
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	bwcvm "bosh-warden-cpi/vm"
)

type VMCloudProperties struct {
	Ports []VMCloudPropertiesPort
}

type VMCloudPropertiesPort struct {
	Host      alwaysString // eg 80, 1000:2000
	Container alwaysString // eg "", 80, 1000:2000
	Protocol  string       // eg "", tcp
	// todo load balancing rules?
}

type alwaysString string

func (s *alwaysString) UnmarshalJSON(data []byte) error {
	*s = alwaysString(strings.TrimPrefix(strings.TrimSuffix(string(data), `"`), `"`))
	return nil
}

func (cp VMCloudProperties) AsVMProps() (bwcvm.VMProps, error) {
	mappings := []bwcvm.PortMapping{}

	for i, p := range cp.Ports {
		mapping, err := cp.portMapping(p)
		if err != nil {
			return bwcvm.VMProps{}, bosherr.WrapErrorf(err, "Validating ports[%v]", i)
		}

		mappings = append(mappings, mapping)
	}

	return bwcvm.VMProps{PortMappings: mappings}, nil
}

func (cp VMCloudProperties) portMapping(p VMCloudPropertiesPort) (bwcvm.PortMapping, error) {
	host, err := bwcvm.NewPortRangeFromString(string(p.Host))
	if err != nil {
		return bwcvm.PortMapping{}, err
	}

	container := host

	if p.Container != "" {
		container, err = bwcvm.NewPortRangeFromString(string(p.Container))
		if err != nil {
			return bwcvm.PortMapping{}, err
		}
	}

	if p.Protocol == "" {
		p.Protocol = "tcp"
	}

	return bwcvm.NewPortMapping(host, container, p.Protocol)
}
