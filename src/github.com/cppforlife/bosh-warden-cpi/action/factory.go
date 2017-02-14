package action

import (
	wrdnclient "github.com/cloudfoundry-incubator/garden/client"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/cppforlife/bosh-cpi-go/apiv1"

	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
	bwcutil "github.com/cppforlife/bosh-warden-cpi/util"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type Factory struct {
	stemcellImporter bwcstem.Importer
	stemcellFinder   bwcstem.Finder

	vmCreator bwcvm.Creator
	vmFinder  bwcvm.Finder

	diskCreator bwcdisk.Creator
	diskFinder  bwcdisk.Finder
}

type CPI struct {
	InfoMethod

	CreateStemcellMethod
	DeleteStemcellMethod

	CreateVMMethod
	DeleteVMMethod
	HasVMMethod
	RebootVMMethod
	SetVMMetadataMethod
	GetDisksMethod

	CreateDiskMethod
	DeleteDiskMethod
	AttachDiskMethod
	DetachDiskMethod
	HasDiskMethod
}

func NewFactory(
	wardenClient wrdnclient.Client,
	fs boshsys.FileSystem,
	cmdRunner boshsys.CmdRunner,
	uuidGen boshuuid.Generator,
	compressor boshcmd.Compressor,
	opts FactoryOpts,
	logger boshlog.Logger,
) Factory {
	stemcellImporter := bwcstem.NewFSImporter(
		opts.StemcellsDir, fs, uuidGen, compressor, logger)

	stemcellFinder := bwcstem.NewFSFinder(opts.StemcellsDir, fs, logger)

	ports := bwcvm.NewIPTablesPorts(cmdRunner)
	sleeper := bwcutil.RealSleeper{}

	hostBindMounts := bwcvm.NewFSHostBindMounts(
		opts.HostEphemeralBindMountsDir, opts.HostPersistentBindMountsDir,
		sleeper, fs, cmdRunner, logger)

	guestBindMounts := bwcvm.NewFSGuestBindMounts(
		opts.GuestEphemeralBindMountPath, opts.GuestPersistentBindMountsDir, logger)

	systemResolvConfProvider := func() (bwcvm.ResolvConf, error) {
		return bwcvm.NewSystemResolvConfFromPath(fs)
	}

	metadataService := bwcvm.NewMetadataService(
		opts.AgentEnvService, opts.Registry, logger)

	agentEnvServiceFactory := bwcvm.NewWardenAgentEnvServiceFactory(
		opts.AgentEnvService, opts.Registry, logger)

	vmCreator := bwcvm.NewWardenCreator(
		uuidGen, wardenClient, metadataService, agentEnvServiceFactory, ports,
		hostBindMounts, guestBindMounts, systemResolvConfProvider, opts.Agent, logger)

	vmFinder := bwcvm.NewWardenFinder(
		wardenClient, agentEnvServiceFactory, ports, hostBindMounts, guestBindMounts, logger)

	diskFactory := bwcdisk.NewFSFactory(opts.DisksDir, fs, uuidGen, cmdRunner, logger)

	return Factory{
		stemcellImporter,
		stemcellFinder,
		vmCreator,
		vmFinder,
		diskFactory,
		diskFactory,
	}
}

func (f Factory) New(_ apiv1.CallContext) (apiv1.CPI, error) {
	return CPI{
		NewInfoMethod(),

		NewCreateStemcellMethod(f.stemcellImporter),
		NewDeleteStemcellMethod(f.stemcellFinder),

		NewCreateVMMethod(f.stemcellFinder, f.vmCreator),
		NewDeleteVMMethod(f.vmFinder),
		NewHasVMMethod(f.vmFinder),
		NewRebootVMMethod(),
		NewSetVMMetadataMethod(),
		NewGetDisksMethod(f.vmFinder),

		NewCreateDiskMethod(f.diskCreator),
		NewDeleteDiskMethod(f.diskFinder),
		NewAttachDiskMethod(f.vmFinder, f.diskFinder),
		NewDetachDiskMethod(f.vmFinder, f.diskFinder),
		NewHasDiskMethod(f.diskFinder),
	}, nil
}
