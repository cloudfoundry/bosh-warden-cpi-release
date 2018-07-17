package vm

import (
	"strings"

	wrdn "code.cloudfoundry.org/garden"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/cppforlife/bosh-cpi-go/apiv1"

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

	agentOptions apiv1.AgentOptions
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
	agentOptions apiv1.AgentOptions,
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

func (c WardenCreator) Create(
	agentID apiv1.AgentID, stemcell bwcstem.Stemcell, props VMProps,
	networks apiv1.Networks, env apiv1.VMEnv) (VM, error) {

	idStr, err := c.uuidGen.Generate()
	if err != nil {
		return WardenVM{}, bosherr.WrapError(err, "Generating VM id")
	}

	id := apiv1.NewVMCID(idStr)

	networkIPCIDR, err := c.resolveNetworkIPCIDR(networks)
	if err != nil {
		return WardenVM{}, err
	}

	systemResolvConf, err := c.systemResolvConfProvider()
	if err != nil {
		return WardenVM{}, err
	}

	networks.BackfillDefaultDNS(systemResolvConf.Nameservers)

	for _, net := range networks {
		net.SetPreconfigured()
	}

	hostEphemeralBindMountPath, hostPersistentBindMountsDir, err := c.makeHostBindMounts(id)
	if err != nil {
		return WardenVM{}, err
	}

	containerSpec := wrdn.ContainerSpec{
		Handle:     id.AsString(),
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

	agentEnv := apiv1.AgentEnvFactory{}.ForVM(agentID, id, networks, env, c.agentOptions)
	agentEnv.AttachSystemDisk(apiv1.NewDiskHintFromString(""))

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

	vm := NewWardenVM(
		id, c.wardenClient, agentEnvService,
		c.ports, c.hostBindMounts, c.guestBindMounts, c.logger, true)

	return vm, nil
}

func (c WardenCreator) resolveNetworkIPCIDR(networks apiv1.Networks) (string, error) {
	var network apiv1.Network

	if len(networks) == 0 {
		return "", bosherr.Error("Expected exactly one network; received zero")
	}

	network = networks.Default()

	if network.IsDynamic() {
		return "", nil
	}

	return network.IPWithSubnetMask(), nil
}

func (c WardenCreator) makeHostBindMounts(id apiv1.VMCID) (string, string, error) {
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
		Path: "/bin/bash",
		User: "root",
		Args: []string{
			"-c",
			strings.Join([]string{
				"umount /etc/resolv.conf",
				"umount /etc/hosts",
				"umount /etc/hostname",
				"rm -rf /var/vcap/data/sys",
				"mkdir -p /var/vcap/data/sys",
				"exec env -i /usr/sbin/runsvdir-start",
			}, "\n"),
		},
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
		c.logger.Error("WardenCreator", "Failed destroying container '%s': %s", container.Handle(), err.Error())
	}
}
