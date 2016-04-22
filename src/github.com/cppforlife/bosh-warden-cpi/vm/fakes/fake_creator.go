package fakes

import (
	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type FakeCreator struct {
	CreateAgentID     string
	CreateStemcell    bwcstem.Stemcell
	CreateNetworks    bwcvm.Networks
	CreateEnvironment bwcvm.Environment
	CreateCloudProperties bwcvm.CloudProperties
	CreateVM          bwcvm.VM
	CreateErr         error
}

func (c *FakeCreator) Create(agentID string, stemcell bwcstem.Stemcell, networks bwcvm.Networks, cloudProperties bwcvm.CloudProperties, env bwcvm.Environment) (bwcvm.VM, error) {
	c.CreateAgentID = agentID
	c.CreateStemcell = stemcell
	c.CreateNetworks = networks
	c.CreateCloudProperties = cloudProperties
	c.CreateEnvironment = env
	return c.CreateVM, c.CreateErr
}
