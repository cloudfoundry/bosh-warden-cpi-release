package vm

import (
	wrdn "github.com/cloudfoundry-incubator/garden"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"

	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
)

type WardenCreator struct {
	uuidGen boshuuid.Generator

	wardenClient           wrdn.Client
	metadataService        MetadataService
	agentEnvServiceFactory AgentEnvServiceFactory

	ports           Ports
	hostBindMounts  HostBindMounts
	guestBindMounts GuestBindMounts

	systemResolvConfProvider func() (ResolvConf, error)

	agentOptions AgentOptions
	logger       boshlog.Logger
}

func NewWardenCreator(
	uuidGen boshuuid.Generator,
	wardenClient wrdn.Client,
	metadataService MetadataService,
	agentEnvServiceFactory AgentEnvServiceFactory,
	ports Ports,
	hostBindMounts HostBindMounts,
	guestBindMounts GuestBindMounts,
	systemResolvConfProvider func() (ResolvConf, error),
	agentOptions AgentOptions,
	logger boshlog.Logger,
) WardenCreator {
	return WardenCreator{
		uuidGen: uuidGen,

		wardenClient:           wardenClient,
		metadataService:        metadataService,
		agentEnvServiceFactory: agentEnvServiceFactory,

		ports:           ports,
		hostBindMounts:  hostBindMounts,
		guestBindMounts: guestBindMounts,

		systemResolvConfProvider: systemResolvConfProvider,

		agentOptions: agentOptions,
		logger:       logger,
	}
}

func (c WardenCreator) Create(agentID string, stemcell bwcstem.Stemcell, props VMProps, networks Networks, env Environment) (VM, error) {
	id, err := c.uuidGen.Generate()
	if err != nil {
		return WardenVM{}, bosherr.WrapError(err, "Generating VM id")
	}

	networkIPCIDR, err := c.resolveNetworkIPCIDR(networks)
	if err != nil {
		return WardenVM{}, err
	}

	systemResolvConf, err := c.systemResolvConfProvider()
	if err != nil {
		return WardenVM{}, err
	}

	networks = networks.BackfillDefaultDNS(systemResolvConf.Nameservers)

	hostEphemeralBindMountPath, hostPersistentBindMountsDir, err := c.makeHostBindMounts(id)
	if err != nil {
		return WardenVM{}, err
	}

	containerSpec := wrdn.ContainerSpec{
		Handle:     id,
		RootFSPath: stemcell.DirPath(),
		Network:    networkIPCIDR,
		BindMounts: []wrdn.BindMount{
			wrdn.BindMount{
				SrcPath: hostEphemeralBindMountPath,
				DstPath: c.guestBindMounts.MakeEphemeral(),
				Mode:    wrdn.BindMountModeRW,
				Origin:  wrdn.BindMountOriginHost,
			},
			wrdn.BindMount{
				SrcPath: hostPersistentBindMountsDir,
				DstPath: c.guestBindMounts.MakePersistent(),
				Mode:    wrdn.BindMountModeRW,
				Origin:  wrdn.BindMountOriginHost,
			},
		},
		Properties: wrdn.Properties{},
		Privileged: true,
	}

	c.logger.Debug("WardenCreator", "Creating container with spec %#v", containerSpec)

	container, err := c.wardenClient.Create(containerSpec)
	if err != nil {
		return WardenVM{}, bosherr.WrapError(err, "Creating container")
	}

	info, err := container.Info()
	if err != nil {
		c.cleanUpContainer(container)
		return WardenVM{}, bosherr.WrapError(err, "Getting container info")
	}

	err = c.ports.Forward(id, info.ContainerIP, props.PortMappings)
	if err != nil {
		c.cleanUpContainer(container)
		return WardenVM{}, bosherr.WrapError(err, "Forwarding host ports")
	}

	agentEnv := NewAgentEnvForVM(agentID, id, networks, env, c.agentOptions)

	wardenFileService := NewWardenFileService(container, c.logger)
	agentEnvService := c.agentEnvServiceFactory.New(wardenFileService, id)

	err = agentEnvService.Update(agentEnv)
	if err != nil {
		c.cleanUpContainer(container)
		return WardenVM{}, bosherr.WrapError(err, "Updating container's agent env")
	}

	err = c.metadataService.Save(wardenFileService, id)
	if err != nil {
		c.cleanUpContainer(container)
		return WardenVM{}, bosherr.WrapError(err, "Updating container's metadata")
	}

	err = c.startAgentInContainer(container)
	if err != nil {
		c.cleanUpContainer(container)
		return WardenVM{}, err
	}

	vm := NewWardenVM(id, c.wardenClient, agentEnvService, c.ports, c.hostBindMounts, c.guestBindMounts, c.logger, true)

	return vm, nil
}

func (c WardenCreator) resolveNetworkIPCIDR(networks Networks) (string, error) {
	var network Network

	if len(networks) == 0 {
		return "", bosherr.Error("Expected exactly one network; received zero")
	}

	network = networks.Default()

	if network.IsDynamic() {
		return "", nil
	}

	return network.IPWithSubnetMask(), nil
}

func (c WardenCreator) makeHostBindMounts(id string) (string, string, error) {
	ephemeralBindMountPath, err := c.hostBindMounts.MakeEphemeral(id)
	if err != nil {
		return "", "", bosherr.WrapError(err, "Making host ephemeral bind mount path")
	}

	persistentBindMountsDir, err := c.hostBindMounts.MakePersistent(id)
	if err != nil {
		return "", "", bosherr.WrapError(err, "Making host persistent bind mounts dir")
	}

	return ephemeralBindMountPath, persistentBindMountsDir, nil
}

func (c WardenCreator) startAgentInContainer(container wrdn.Container) error {
	processSpec := wrdn.ProcessSpec{
		Path: "/usr/sbin/runsvdir-start",
		User: "root",
	}

	// Do not Wait() for the process to finish
	_, err := container.Run(processSpec, wrdn.ProcessIO{})
	if err != nil {
		return bosherr.WrapError(err, "Running BOSH Agent in container")
	}

	return nil
}

func (c WardenCreator) cleanUpContainer(container wrdn.Container) {
	// false is to kill immediately
	err := container.Stop(false)
	if err != nil {
		c.logger.Error("WardenCreator", "Failed destroying container '%s': %s", container.Handle, err.Error())
	}
}
