package vm_test

import (
	"errors"
	"strings"

	wrdn "code.cloudfoundry.org/garden"
	wrdnclient "code.cloudfoundry.org/garden/client"
	fakewrdnconn "code.cloudfoundry.org/garden/client/connection/connectionfakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakestem "github.com/cppforlife/bosh-warden-cpi/stemcell/fakes"
	. "github.com/cppforlife/bosh-warden-cpi/vm"
	fakevm "github.com/cppforlife/bosh-warden-cpi/vm/fakes"
)

var _ = Describe("WardenCreator", func() {
	var (
		wardenConn   *fakewrdnconn.FakeConnection
		wardenClient wrdnclient.Client

		uuidGen                *fakeuuid.FakeGenerator
		fakeMetadataService    *fakevm.FakeMetadataService
		agentEnvServiceFactory *fakevm.FakeAgentEnvServiceFactory
		ports                  *fakevm.FakePorts
		hostBindMounts         *fakevm.FakeHostBindMounts
		guestBindMounts        *fakevm.FakeGuestBindMounts

		systemResolvConfProviderConf ResolvConf
		systemResolvConfProviderErr  error
		agentOptions                 apiv1.AgentOptions

		logger  boshlog.Logger
		creator WardenCreator
	)

	BeforeEach(func() {
		wardenConn = &fakewrdnconn.FakeConnection{}
		wardenClient = wrdnclient.New(wardenConn)

		uuidGen = &fakeuuid.FakeGenerator{}
		fakeMetadataService = fakevm.NewFakeMetadataService()
		agentEnvServiceFactory = &fakevm.FakeAgentEnvServiceFactory{}
		ports = &fakevm.FakePorts{}
		hostBindMounts = &fakevm.FakeHostBindMounts{}
		guestBindMounts = &fakevm.FakeGuestBindMounts{
			EphemeralBindMountPath:  "/fake-guest-ephemeral-bind-mount-path",
			PersistentBindMountsDir: "/fake-guest-persistent-bind-mounts-dir",
		}

		systemResolvConfProviderConf = ResolvConf{}
		systemResolvConfProviderErr = nil
		agentOptions = apiv1.AgentOptions{Mbus: "fake-mbus"}

		resolvProvider := func() (ResolvConf, error) {
			return systemResolvConfProviderConf, systemResolvConfProviderErr
		}
		logger = boshlog.NewLogger(boshlog.LevelNone)

		creator = NewWardenCreator(
			uuidGen, wardenClient, fakeMetadataService, agentEnvServiceFactory,
			ports, hostBindMounts, guestBindMounts, resolvProvider, agentOptions, logger)
	})

	Describe("Create", func() {
		var (
			stemcell *fakestem.FakeStemcell
			networks apiv1.Networks
			env      apiv1.VMEnv
		)

		BeforeEach(func() {
			stemcell = fakestem.NewFakeStemcellWithPath(
				apiv1.NewStemcellCID("fake-stemcell-id"),
				"/fake-stemcell-path",
			)

			networks = apiv1.Networks{
				"fake-net-name": apiv1.NewNetwork(apiv1.NetworkOpts{}),
			}

			env = apiv1.NewVMEnv(map[string]interface{}{"fake-env-key": "fake-env-value"})
		})

		It("returns created vm", func() {
			uuidGen.GeneratedUUID = "fake-vm-id"

			agentEnvService := &fakevm.FakeAgentEnvService{}
			agentEnvServiceFactory.NewAgentEnvService = agentEnvService

			expectedVM := NewWardenVM(
				apiv1.NewVMCID("fake-vm-id"), wardenClient, agentEnvService,
				ports, hostBindMounts, guestBindMounts, logger, true)

			vm, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
			Expect(err).ToNot(HaveOccurred())
			Expect(vm).To(Equal(expectedVM))
		})

		Context("when generating VM id succeeds", func() {
			BeforeEach(func() {
				uuidGen.GeneratedUUID = "fake-vm-id"
			})

			It("returns error if zero networks are provided", func() {
				vm, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, apiv1.Networks{}, env)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Expected exactly one network; received zero"))
				Expect(vm).To(Equal(WardenVM{}))
			})

			It("does not return error if more than one network is provided so that warden CPI can be used for testing multiple networks even though garden only supports single network", func() {
				networks = apiv1.Networks{
					"fake-net1": apiv1.NewNetwork(apiv1.NetworkOpts{}),
					"fake-net2": apiv1.NewNetwork(apiv1.NetworkOpts{}),
				}

				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns error if system resolv conf cannot be obtained", func() {
				systemResolvConfProviderErr = errors.New("fake-err")

				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("backfills DNS for the default networks if not set", func() {
				systemResolvConfProviderConf = ResolvConf{Nameservers: []string{"8.8.8.8"}}

				agentEnvService := &fakevm.FakeAgentEnvService{}
				agentEnvServiceFactory.NewAgentEnvService = agentEnvService

				networks["fake-net-name"] = apiv1.NewNetwork(apiv1.NetworkOpts{Default: []string{"dns"}})

				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).ToNot(HaveOccurred())

				// todo Expect(agentEnvService.UpdateAgentEnv.Networks["fake-net-name"].DNS()).To(Equal([]string{"8.8.8.8"}))
			})

			It("creates one container with generated VM id", func() {
				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).ToNot(HaveOccurred())

				count := wardenConn.CreateCallCount()
				Expect(count).To(Equal(1))

				containerSpec := wardenConn.CreateArgsForCall(0)
				Expect(containerSpec.Handle).To(Equal("fake-vm-id"))
			})

			It("creates container with stemcell as its root fs", func() {
				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).ToNot(HaveOccurred())

				containerSpec := wardenConn.CreateArgsForCall(0)
				Expect(containerSpec.RootFSPath).To(Equal("/fake-stemcell-path"))
			})

			It("creates container with bind mounted ephemeral disk and persistent root location", func() {
				hostBindMounts.MakeEphemeralPath = "/fake-host-ephemeral-bind-mount-path"
				hostBindMounts.MakePersistentPath = "/fake-host-persistent-bind-mounts-dir"

				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).ToNot(HaveOccurred())

				containerSpec := wardenConn.CreateArgsForCall(0)
				Expect(containerSpec.BindMounts).To(Equal(
					[]wrdn.BindMount{
						wrdn.BindMount{
							SrcPath: "/fake-host-ephemeral-bind-mount-path",
							DstPath: "/fake-guest-ephemeral-bind-mount-path",
							Mode:    wrdn.BindMountModeRW,
							Origin:  wrdn.BindMountOriginHost,
						},
						wrdn.BindMount{
							SrcPath: "/fake-host-persistent-bind-mounts-dir",
							DstPath: "/fake-guest-persistent-bind-mounts-dir",
							Mode:    wrdn.BindMountModeRW,
							Origin:  wrdn.BindMountOriginHost,
						},
					},
				))

				Expect(hostBindMounts.MakeEphemeralID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
				Expect(hostBindMounts.MakePersistentID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
			})

			It("returns error if making host ephemeral bind mount fails", func() {
				hostBindMounts.MakeEphemeralErr = errors.New("fake-make-ephemeral-err")

				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-make-ephemeral-err"))
			})

			It("returns error if making host persistent bind mount fails", func() {
				hostBindMounts.MakePersistentErr = errors.New("fake-make-persistent-err")

				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-make-persistent-err"))
			})

			It("creates container with IP address if network is not dynamic", func() {
				networks["fake-net-name"] = apiv1.NewNetwork(apiv1.NetworkOpts{
					Type:    "not-dynamic",
					IP:      "10.244.0.0",
					Netmask: "255.255.255.0",
				})

				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).ToNot(HaveOccurred())

				containerSpec := wardenConn.CreateArgsForCall(0)
				Expect(containerSpec.Network).To(Equal("10.244.0.0/24"))
			})

			It("creates container without IP address if network is dynamic", func() {
				networks["fake-net-name"] = apiv1.NewNetwork(apiv1.NetworkOpts{
					Type: "dynamic",
					IP:   "fake-ip", // is not usually set
				})

				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).ToNot(HaveOccurred())

				containerSpec := wardenConn.CreateArgsForCall(0)
				Expect(containerSpec.Network).To(BeEmpty()) // fake-ip is not used
			})

			It("creates container without properties", func() {
				_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).ToNot(HaveOccurred())

				containerSpec := wardenConn.CreateArgsForCall(0)
				Expect(containerSpec.Properties).To(Equal(wrdn.Properties{}))
			})

			Context("when creating container succeeds", func() {
				var (
					agentEnvService *fakevm.FakeAgentEnvService
				)

				BeforeEach(func() {
					agentEnvService = &fakevm.FakeAgentEnvService{}
					agentEnvServiceFactory.NewAgentEnvService = agentEnvService
					wardenConn.CreateReturns("fake-vm-id", nil)
				})

				It("updates container's agent env", func() {
					_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
					Expect(err).ToNot(HaveOccurred())

					expectedAgentEnv := apiv1.AgentEnvFactory{}.ForVM(
						apiv1.NewAgentID("fake-agent-id"), apiv1.NewVMCID("fake-vm-id"), networks, env, agentOptions)
					expectedAgentEnv.AttachSystemDisk(apiv1.NewDiskHintFromString(""))

					Expect(agentEnvServiceFactory.NewWardenFileService).ToNot(BeNil()) // todo
					Expect(agentEnvServiceFactory.NewInstanceID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
					Expect(agentEnvService.UpdateAgentEnv).To(Equal(expectedAgentEnv))
				})

				It("saves metadata", func() {
					wardenConn.CreateReturns("fake-container-handle", nil)
					_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeMetadataService.Saved).To(BeTrue())
					Expect(fakeMetadataService.SaveInstanceID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
				})

				ItDestroysContainer := func(errMsg string) {
					It("destroys created container", func() {
						_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
						Expect(err).To(HaveOccurred())

						count := wardenConn.StopCallCount()
						Expect(count).To(Equal(1))

						handle, force := wardenConn.StopArgsForCall(0)
						Expect(handle).To(Equal("fake-vm-id"))
						Expect(force).To(BeFalse())
					})

					Context("when destroying created container fails", func() {
						BeforeEach(func() {
							wardenConn.StopReturns(errors.New("fake-stop-err"))
						})

						It("returns running error and not destroy error", func() {
							vm, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring(errMsg))
							Expect(vm).To(Equal(WardenVM{}))
						})
					})
				}

				Context("when container's agent env succeeds", func() {
					It("starts BOSH Agent in the container", func() {
						_, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
						Expect(err).ToNot(HaveOccurred())

						count := wardenConn.RunCallCount()
						Expect(count).To(Equal(1))

						expectedProcessSpec := wrdn.ProcessSpec{
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

						handle, processSpec, processIO := wardenConn.RunArgsForCall(0)
						Expect(handle).To(Equal("fake-vm-id"))
						Expect(processSpec).To(Equal(expectedProcessSpec))
						Expect(processIO).To(Equal(wrdn.ProcessIO{}))
					})

					Context("when BOSH Agent fails to start", func() {
						BeforeEach(func() {
							wardenConn.RunReturns(nil, errors.New("fake-run-err"))
						})

						It("returns error if starting BOSH Agent fails", func() {
							vm, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("fake-run-err"))
							Expect(vm).To(Equal(WardenVM{}))
						})

						ItDestroysContainer("fake-run-err")
					})
				})

				Context("when container's agent env update fails", func() {
					BeforeEach(func() {
						agentEnvService.UpdateErr = errors.New("fake-update-err")
					})

					It("returns error because BOSH Agent will fail to start without agent env", func() {
						vm, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-update-err"))
						Expect(vm).To(Equal(WardenVM{}))
					})

					ItDestroysContainer("fake-update-err")
				})
			})

			Context("when creating container fails", func() {
				BeforeEach(func() {
					wardenConn.CreateReturns("fake-vm-id", errors.New("fake-create-err"))
				})

				It("returns error if creating container fails", func() {
					vm, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-create-err"))
					Expect(vm).To(Equal(WardenVM{}))
				})
			})
		})

		Context("when generating VM id fails", func() {
			BeforeEach(func() {
				uuidGen.GenerateError = errors.New("fake-generate-err")
			})

			It("returns error if generating VM id fails", func() {
				vm, err := creator.Create(apiv1.NewAgentID("fake-agent-id"), stemcell, VMProps{}, networks, env)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-generate-err"))
				Expect(vm).To(Equal(WardenVM{}))
			})
		})
	})
})
