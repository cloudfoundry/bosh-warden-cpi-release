package action

import (
	wrdnclient "github.com/cloudfoundry-incubator/garden/client"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"

	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
	bwcutil "github.com/cppforlife/bosh-warden-cpi/util"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type concreteFactory struct {
	availableActions map[string]Action
}

func NewConcreteFactory(
	wardenClient wrdnclient.Client,
	fs boshsys.FileSystem,
	cmdRunner boshsys.CmdRunner,
	uuidGen boshuuid.Generator,
	compressor boshcmd.Compressor,
	sleeper bwcutil.Sleeper,
	options ConcreteFactoryOptions,
	logger boshlog.Logger,
) concreteFactory {
	stemcellImporter := bwcstem.NewFSImporter(
		options.StemcellsDir,
		fs,
		uuidGen,
		compressor,
		logger,
	)

	stemcellFinder := bwcstem.NewFSFinder(options.StemcellsDir, fs, logger)

	ports := bwcvm.NewIPTablesPorts(cmdRunner)

	hostBindMounts := bwcvm.NewFSHostBindMounts(
		options.HostEphemeralBindMountsDir,
		options.HostPersistentBindMountsDir,
		sleeper,
		fs,
		cmdRunner,
		logger,
	)

	guestBindMounts := bwcvm.NewFSGuestBindMounts(
		options.GuestEphemeralBindMountPath,
		options.GuestPersistentBindMountsDir,
		logger,
	)

	systemResolvConfProvider := func() (bwcvm.ResolvConf, error) {
		return bwcvm.NewSystemResolvConfFromPath(fs)
	}

	metadataService := bwcvm.NewMetadataService(options.AgentEnvService, options.Registry, logger)

	agentEnvServiceFactory := bwcvm.NewWardenAgentEnvServiceFactory(options.AgentEnvService, options.Registry, logger)

	vmCreator := bwcvm.NewWardenCreator(
		uuidGen,
		wardenClient,
		metadataService,
		agentEnvServiceFactory,
		ports,
		hostBindMounts,
		guestBindMounts,
		systemResolvConfProvider,
		options.Agent,
		logger,
	)

	vmFinder := bwcvm.NewWardenFinder(
		wardenClient,
		agentEnvServiceFactory,
		ports,
		hostBindMounts,
		guestBindMounts,
		logger,
	)

	diskCreator := bwcdisk.NewFSCreator(
		options.DisksDir,
		fs,
		uuidGen,
		cmdRunner,
		logger,
	)

	diskFinder := bwcdisk.NewFSFinder(options.DisksDir, fs, logger)

	return concreteFactory{
		availableActions: map[string]Action{
			// Stemcell management
			"create_stemcell": NewCreateStemcell(stemcellImporter),
			"delete_stemcell": NewDeleteStemcell(stemcellFinder),

			// VM management
			"create_vm":          NewCreateVM(stemcellFinder, vmCreator),
			"delete_vm":          NewDeleteVM(vmFinder, hostBindMounts),
			"has_vm":             NewHasVM(vmFinder),
			"reboot_vm":          NewRebootVM(),
			"set_vm_metadata":    NewSetVMMetadata(),
			"configure_networks": NewConfigureNetworks(),

			// Disk management
			"create_disk": NewCreateDisk(diskCreator),
			"delete_disk": NewDeleteDisk(diskFinder),
			"attach_disk": NewAttachDisk(vmFinder, diskFinder),
			"detach_disk": NewDetachDisk(vmFinder, diskFinder),

			// Not implemented:
			//   current_vm_id
			//   snapshot_disk
			//   delete_snapshot
			//   get_disks
			//   ping
		},
	}
}

func (f concreteFactory) Create(method string) (Action, error) {
	action, found := f.availableActions[method]
	if !found {
		return nil, bosherr.Errorf("Could not create action with method '%s'", method)
	}

	return action, nil
}
