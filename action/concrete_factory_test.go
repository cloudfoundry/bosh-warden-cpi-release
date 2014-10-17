package action_test

import (
	fakewrdnclient "github.com/cloudfoundry-incubator/garden/client/fake_warden_client"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	fakecmd "github.com/cloudfoundry/bosh-agent/platform/commands/fakes"
	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-agent/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/action"
	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
	bwcutil "github.com/cppforlife/bosh-warden-cpi/util"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

var _ = Describe("concreteFactory", func() {
	var (
		wardenClient *fakewrdnclient.FakeClient
		fs           *fakesys.FakeFileSystem
		cmdRunner    *fakesys.FakeCmdRunner
		uuidGen      *fakeuuid.FakeGenerator
		compressor   *fakecmd.FakeCompressor
		sleeper      bwcutil.Sleeper
		logger       boshlog.Logger

		options = ConcreteFactoryOptions{
			StemcellsDir: "/tmp/stemcells",
			DisksDir:     "/tmp/disks",

			HostEphemeralBindMountsDir:  "/tmp/host-ephemeral-bind-mounts-dir",
			HostPersistentBindMountsDir: "/tmp/host-persistent-bind-mounts-dir",

			GuestEphemeralBindMountPath:  "/tmp/guest-ephemeral-bind-mount-path",
			GuestPersistentBindMountsDir: "/tmp/guest-persistent-bind-mounts-dir",

			AgentEnvService: "registry",
			Registry: bwcvm.RegistryOptions{
				Username: "fake-user",
				Password: "fake-password",
				Host:     "fake-host",
				Port:     1234,
			},
		}

		factory Factory
	)

	var (
		agentEnvServiceFactory bwcvm.AgentEnvServiceFactory

		hostBindMounts  bwcvm.FSHostBindMounts
		guestBindMounts bwcvm.FSGuestBindMounts

		stemcellFinder bwcstem.Finder
		vmFinder       bwcvm.Finder
		diskFinder     bwcdisk.Finder
	)

	BeforeEach(func() {
		wardenClient = fakewrdnclient.New()
		fs = fakesys.NewFakeFileSystem()
		cmdRunner = fakesys.NewFakeCmdRunner()
		uuidGen = &fakeuuid.FakeGenerator{}
		compressor = fakecmd.NewFakeCompressor()
		sleeper = bwcutil.RealSleeper{}
		logger = boshlog.NewLogger(boshlog.LevelNone)

		factory = NewConcreteFactory(
			wardenClient,
			fs,
			cmdRunner,
			uuidGen,
			compressor,
			sleeper,
			options,
			logger,
		)
	})

	BeforeEach(func() {
		hostBindMounts = bwcvm.NewFSHostBindMounts(
			"/tmp/host-ephemeral-bind-mounts-dir",
			"/tmp/host-persistent-bind-mounts-dir",
			sleeper,
			fs,
			cmdRunner,
			logger,
		)

		guestBindMounts = bwcvm.NewFSGuestBindMounts(
			"/tmp/guest-ephemeral-bind-mount-path",
			"/tmp/guest-persistent-bind-mounts-dir",
			logger,
		)

		agentEnvServiceFactory = bwcvm.NewWardenAgentEnvServiceFactory(options.AgentEnvService, options.Registry, logger)

		stemcellFinder = bwcstem.NewFSFinder("/tmp/stemcells", fs, logger)

		vmFinder = bwcvm.NewWardenFinder(
			wardenClient,
			agentEnvServiceFactory,
			hostBindMounts,
			guestBindMounts,
			logger,
		)

		diskFinder = bwcdisk.NewFSFinder("/tmp/disks", fs, logger)
	})

	It("returns error if action cannot be created", func() {
		action, err := factory.Create("fake-unknown-action")
		Expect(err).To(HaveOccurred())
		Expect(action).To(BeNil())
	})

	It("create_stemcell", func() {
		stemcellImporter := bwcstem.NewFSImporter(
			"/tmp/stemcells",
			fs,
			uuidGen,
			compressor,
			logger,
		)

		action, err := factory.Create("create_stemcell")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewCreateStemcell(stemcellImporter)))
	})

	It("delete_stemcell", func() {
		action, err := factory.Create("delete_stemcell")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewDeleteStemcell(stemcellFinder)))
	})

	It("create_vm", func() {
		vmCreator := bwcvm.NewWardenCreator(
			uuidGen,
			wardenClient,
			agentEnvServiceFactory,
			hostBindMounts,
			guestBindMounts,
			options.Agent,
			logger,
		)

		action, err := factory.Create("create_vm")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewCreateVM(stemcellFinder, vmCreator)))
	})

	It("delete_vm", func() {
		action, err := factory.Create("delete_vm")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewDeleteVM(vmFinder, hostBindMounts)))
	})

	It("has_vm", func() {
		action, err := factory.Create("has_vm")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewHasVM(vmFinder)))
	})

	It("reboot_vm", func() {
		action, err := factory.Create("reboot_vm")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewRebootVM()))
	})

	It("set_vm_metadata", func() {
		action, err := factory.Create("set_vm_metadata")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewSetVMMetadata()))
	})

	It("configure_networks", func() {
		action, err := factory.Create("configure_networks")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewConfigureNetworks()))
	})

	It("create_disk", func() {
		diskCreator := bwcdisk.NewFSCreator(
			"/tmp/disks",
			fs,
			uuidGen,
			cmdRunner,
			logger,
		)

		action, err := factory.Create("create_disk")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewCreateDisk(diskCreator)))
	})

	It("delete_disk", func() {
		action, err := factory.Create("delete_disk")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewDeleteDisk(diskFinder)))
	})

	It("attach_disk", func() {
		action, err := factory.Create("attach_disk")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewAttachDisk(vmFinder, diskFinder)))
	})

	It("detach_disk", func() {
		action, err := factory.Create("detach_disk")
		Expect(err).ToNot(HaveOccurred())
		Expect(action).To(Equal(NewDetachDisk(vmFinder, diskFinder)))
	})

	It("returns error because CPI machine is not self-aware if action is current_vm_id", func() {
		action, err := factory.Create("current_vm_id")
		Expect(err).To(HaveOccurred())
		Expect(action).To(BeNil())
	})

	It("returns error because snapshotting is not implemented if action is snapshot_disk", func() {
		action, err := factory.Create("snapshot_disk")
		Expect(err).To(HaveOccurred())
		Expect(action).To(BeNil())
	})

	It("returns error because snapshotting is not implemented if action is delete_snapshot", func() {
		action, err := factory.Create("delete_snapshot")
		Expect(err).To(HaveOccurred())
		Expect(action).To(BeNil())
	})

	It("returns error since CPI should not keep state if action is get_disks", func() {
		action, err := factory.Create("get_disks")
		Expect(err).To(HaveOccurred())
		Expect(action).To(BeNil())
	})

	It("returns error because ping is not official CPI method if action is ping", func() {
		action, err := factory.Create("ping")
		Expect(err).To(HaveOccurred())
		Expect(action).To(BeNil())
	})
})
