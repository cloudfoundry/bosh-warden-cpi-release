package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"

	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type CreateVMMethod struct {
	stemcellFinder bwcstem.Finder
	vmCreator      bwcvm.Creator
	versions       apiv1.ApiVersions
}

func NewCreateVMMethod(stemcellFinder bwcstem.Finder, vmCreator bwcvm.Creator, versions apiv1.ApiVersions) CreateVMMethod {
	return CreateVMMethod{
		versions:       versions,
		stemcellFinder: stemcellFinder,
		vmCreator:      vmCreator,
	}
}

func (a CreateVMMethod) CreateVM(
	agentID apiv1.AgentID, stemcellCID apiv1.StemcellCID,
	cloudProps apiv1.VMCloudProps, networks apiv1.Networks,
	associatedDiskCIDs []apiv1.DiskCID, env apiv1.VMEnv) (interface{}, error) {

	stemcell, found, err := a.stemcellFinder.Find(stemcellCID)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Finding stemcell '%s'", stemcellCID)
	}

	if !found {
		return nil, bosherr.Errorf("Expected to find stemcell '%s'", stemcellCID)
	}

	var customCloudProps VMCloudProperties

	err = cloudProps.As(&customCloudProps)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Parsing VM cloud properties")
	}

	vmProps, err := customCloudProps.AsVMProps()
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Validating 'ports' configuration")
	}

	vm, err := a.vmCreator.Create(agentID, stemcell, vmProps, networks, env)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Creating VM with agent ID '%s'", agentID)
	}
	if a.versions.Contract == 2 {
		return []interface{}{vm.ID(), networks}, nil
	} else {
		return vm.ID(), nil
	}
}
