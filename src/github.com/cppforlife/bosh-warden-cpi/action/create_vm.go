package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type CreateVM struct {
	stemcellFinder bwcstem.Finder
	vmCreator      bwcvm.Creator
}

type VMCloudProperties struct {
	Ports []VMCloudPropertiesPort
}

type VMCloudPropertiesPort struct {
	Host      int
	Container int
	Protocol  string
	// todo load balancing rules?
}

func (cp VMCloudProperties) AsVMProps() (bwcvm.VMProps, error) {
	ports := []bwcvm.VMPropsPort{}

	for i, p := range cp.Ports {
		port, err := cp.port(p)
		if err != nil {
			return bwcvm.VMProps{}, bosherr.WrapErrorf(err, "Validating ports[%i]", i)
		}

		ports = append(ports, port)
	}

	return bwcvm.VMProps{Ports: ports}, nil
}

func (cp VMCloudProperties) port(p VMCloudPropertiesPort) (bwcvm.VMPropsPort, error) {
	port := bwcvm.VMPropsPort{
		Host:      p.Host,
		Container: p.Container,
		Protocol:  p.Protocol,
	}

	if port.Host <= 0 {
		return port, bosherr.Errorf("'host' port must be > 0")
	}

	switch {
	case port.Container == 0:
		port.Container = port.Host
	case port.Container < 0:
		return port, bosherr.Errorf("'container' port must be >= 0")
	}

	switch port.Protocol {
	case "tcp", "udp":
		// valid
	case "":
		port.Protocol = "tcp"
	default:
		return port, bosherr.Errorf("'protocol' must be 'tcp', 'udp' or '' (default tcp)")
	}

	return port, nil
}

type Environment map[string]interface{}

func NewCreateVM(stemcellFinder bwcstem.Finder, vmCreator bwcvm.Creator) CreateVM {
	return CreateVM{
		stemcellFinder: stemcellFinder,
		vmCreator:      vmCreator,
	}
}

func (a CreateVM) Run(agentID string, stemcellCID StemcellCID, cloudProps VMCloudProperties, networks Networks, _ []DiskCID, env Environment) (VMCID, error) {
	stemcell, found, err := a.stemcellFinder.Find(string(stemcellCID))
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Finding stemcell '%s'", stemcellCID)
	}

	if !found {
		return "", bosherr.Errorf("Expected to find stemcell '%s'", stemcellCID)
	}

	vmNetworks := networks.AsVMNetworks()
	vmEnv := bwcvm.Environment(env)

	vmProps, err := cloudProps.AsVMProps()
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Validating 'ports' configuration")
	}

	vm, err := a.vmCreator.Create(agentID, stemcell, vmProps, vmNetworks, vmEnv)
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Creating VM with agent ID '%s'", agentID)
	}

	return VMCID(vm.ID()), nil
}
