package fakes

import (
	apiv1 "github.com/cloudfoundry/bosh-cpi-go/apiv1"

	bwcstem "bosh-warden-cpi/stemcell"
	bwcvm "bosh-warden-cpi/vm"
)

type FakeCreator struct {
	CreateAgentID     apiv1.AgentID
	CreateStemcell    bwcstem.Stemcell
	CreateProps       bwcvm.VMProps
	CreateNetworks    apiv1.Networks
	CreateEnvironment apiv1.VMEnv
	CreateVM          bwcvm.VM
	CreateErr         error
}

func (c *FakeCreator) Create(
	agentID apiv1.AgentID, stemcell bwcstem.Stemcell, props bwcvm.VMProps,
	networks apiv1.Networks, env apiv1.VMEnv) (bwcvm.VM, error) {

	c.CreateAgentID = agentID
	c.CreateProps = props
	c.CreateStemcell = stemcell
	c.CreateNetworks = networks
	c.CreateEnvironment = env

	return c.CreateVM, c.CreateErr
}
