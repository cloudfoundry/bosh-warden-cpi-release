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

type Environment map[string]interface{}

func NewCreateVM(stemcellFinder bwcstem.Finder, vmCreator bwcvm.Creator) CreateVM {
	return CreateVM{
		stemcellFinder: stemcellFinder,
		vmCreator:      vmCreator,
	}
}

func (a CreateVM) Run(agentID string, stemcellCID StemcellCID, cloudProperties VMCloudProperties, networks Networks, _ []DiskCID, env Environment) (VMCID, error) {
	stemcell, found, err := a.stemcellFinder.Find(string(stemcellCID))
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Finding stemcell '%s'", stemcellCID)
	}

	if !found {
		return "", bosherr.Errorf("Expected to find stemcell '%s'", stemcellCID)
	}

	vmNetworks := networks.AsVMNetworks()
	vmCloudProperties := cloudProperties.AsVMCloudProperties()
	vmEnv := bwcvm.Environment(env)

	vm, err := a.vmCreator.Create(agentID, stemcell, vmNetworks, vmCloudProperties, vmEnv)
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Creating VM with agent ID '%s'", agentID)
	}

	return VMCID(vm.ID()), nil
}
