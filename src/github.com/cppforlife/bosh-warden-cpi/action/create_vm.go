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

type VMCloudProperties struct{}

type Environment map[string]interface{}

func NewCreateVM(stemcellFinder bwcstem.Finder, vmCreator bwcvm.Creator) CreateVM {
	return CreateVM{
		stemcellFinder: stemcellFinder,
		vmCreator:      vmCreator,
	}
}

func (a CreateVM) Run(agentID string, stemcellCID StemcellCID, _ VMCloudProperties, networks Networks, _ []DiskCID, env Environment) (VMCID, error) {
	stemcell, found, err := a.stemcellFinder.Find(string(stemcellCID))
	if err != nil {
		return "", bosherr.WrapError(err, "Finding stemcell '%s'", stemcellCID)
	}

	if !found {
		return "", bosherr.New("Expected to find stemcell '%s'", stemcellCID)
	}

	vmNetworks := networks.AsVMNetworks()

	vmEnv := bwcvm.Environment(env)

	vm, err := a.vmCreator.Create(agentID, stemcell, vmNetworks, vmEnv)
	if err != nil {
		return "", bosherr.WrapError(err, "Creating VM with agent ID '%s'", agentID)
	}

	return VMCID(vm.ID()), nil
}
