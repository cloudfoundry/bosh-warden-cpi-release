package vm_test

import (
	"errors"

	wrdnclient "github.com/cloudfoundry-incubator/garden/client"
	fakewrdnconn "github.com/cloudfoundry-incubator/garden/client/connection/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakedisk "github.com/cppforlife/bosh-warden-cpi/disk/fakes"
	. "github.com/cppforlife/bosh-warden-cpi/vm"
	fakevm "github.com/cppforlife/bosh-warden-cpi/vm/fakes"
)

var _ = Describe("WardenVM", func() {
	var (
		wardenConn   *fakewrdnconn.FakeConnection
		wardenClient wrdnclient.Client

		agentEnvService *fakevm.FakeAgentEnvService
		ports           *fakevm.FakePorts
		hostBindMounts  *fakevm.FakeHostBindMounts
		guestBindMounts *fakevm.FakeGuestBindMounts
		logger          boshlog.Logger
		vm              WardenVM
	)

	BeforeEach(func() {
		wardenConn = &fakewrdnconn.FakeConnection{}
		wardenClient = wrdnclient.New(wardenConn)

		agentEnvService = &fakevm.FakeAgentEnvService{}
		ports = &fakevm.FakePorts{}
		hostBindMounts = &fakevm.FakeHostBindMounts{}
		guestBindMounts = &fakevm.FakeGuestBindMounts{
			EphemeralBindMountPath:  "/fake-guest-ephemeral-bind-mount-path",
			PersistentBindMountsDir: "/fake-guest-persistent-bind-mounts-dir",
		}
		logger = boshlog.NewLogger(boshlog.LevelNone)

		vm = NewWardenVM(
			apiv1.NewVMCID("fake-vm-id"), wardenClient, agentEnvService,
			ports, hostBindMounts, guestBindMounts, logger, true)
	})

	Describe("Delete", func() {
		It("destroys container before deleting bind mounts so that they are not marked as busy by the kernel", func() {
			err := vm.Delete()
			Expect(err).ToNot(HaveOccurred())

			Expect(wardenConn.DestroyCallCount()).To(Equal(1))
			Expect(wardenConn.DestroyArgsForCall(0)).To(Equal("fake-vm-id"))
		})

		Context("when destroying container succeeds", func() {
			It("deletes ephemeral bind mount dir", func() {
				err := vm.Delete()
				Expect(err).ToNot(HaveOccurred())

				Expect(hostBindMounts.DeleteEphemeralID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
			})

			Context("when deleting ephemeral bind mount dir succeeds", func() {
				It("deletes persistent bind mounts dir for persistent disks and returns no error", func() {
					err := vm.Delete()
					Expect(err).ToNot(HaveOccurred())

					Expect(hostBindMounts.DeletePersistentID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
				})

				Context("when deleting persistent bind mounts dir fails", func() {
					BeforeEach(func() {
						hostBindMounts.DeletePersistentErr = errors.New("fake-delete-persistent-err")
					})

					It("returns error", func() {
						err := vm.Delete()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-delete-persistent-err"))
					})
				})
			})

			Context("when deleting ephemeral bind mount dir fails", func() {
				BeforeEach(func() {
					hostBindMounts.DeleteEphemeralErr = errors.New("fake-delete-ephemeral-err")
				})

				It("returns error", func() {
					err := vm.Delete()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-delete-ephemeral-err"))
				})
			})
		})

		Context("when destroying container fails", func() {
			BeforeEach(func() {
				wardenConn.DestroyReturns(errors.New("fake-destroy-err"))
			})

			It("returns error", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-destroy-err"))
			})

			It("does not delete ephemeral bind mounts dir", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())

				Expect(hostBindMounts.DeleteEphemeralCalled).To(BeFalse())
			})

			It("does not delete persistent bind mounts dir", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())

				Expect(hostBindMounts.DeletePersistentCalled).To(BeFalse())
			})
		})

		Context("when the container does not exist", func() {
			BeforeEach(func() {
				vm = NewWardenVM(
					apiv1.NewVMCID("fake-vm-id"), wardenClient, nil,
					ports, hostBindMounts, guestBindMounts, logger, false)
			})

			It("deletes ephemeral and persistent bind mount dirs", func() {
				err := vm.Delete()
				Expect(err).ToNot(HaveOccurred())

				Expect(hostBindMounts.DeleteEphemeralID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
				Expect(hostBindMounts.DeletePersistentID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
			})
		})
	})

	Describe("AttachDisk", func() {
		var (
			disk *fakedisk.FakeDisk
		)

		BeforeEach(func() {
			agentEnv := &apiv1.AgentEnvImpl{}
			agentEnvService.FetchAgentEnv = agentEnv

			disk = fakedisk.NewFakeDiskWithPath(apiv1.NewDiskCID("fake-disk-id"), "/fake-disk-path")
		})

		It("tries to fetch agent env", func() {
			err := vm.AttachDisk(disk)
			Expect(err).ToNot(HaveOccurred())

			Expect(agentEnvService.FetchCalled).To(BeTrue())
		})

		Context("when fetching agent env succeeds", func() {
			BeforeEach(func() {
				agentEnv := &apiv1.AgentEnvImpl{}
				agentEnv.AttachPersistentDisk(apiv1.NewDiskCID("fake-disk-id2"), "/fake-hint-path2")
				agentEnvService.FetchAgentEnv = agentEnv
			})

			It("mounts persistent bind mounts dir", func() {
				err := vm.AttachDisk(disk)
				Expect(err).ToNot(HaveOccurred())

				Expect(hostBindMounts.MountPersistentID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
				Expect(hostBindMounts.MountPersistentDiskID).To(Equal(apiv1.NewDiskCID("fake-disk-id")))
				Expect(hostBindMounts.MountPersistentDiskPath).To(Equal("/fake-disk-path"))
			})

			Context("when mounting persistent bind mounts dir succeeds", func() {
				It("updates agent env attaching persistent disk", func() {
					err := vm.AttachDisk(disk)
					Expect(err).ToNot(HaveOccurred())

					afterAgentEnv := &apiv1.AgentEnvImpl{}
					afterAgentEnv.AttachPersistentDisk(apiv1.NewDiskCID("fake-disk-id2"), "/fake-hint-path2")
					afterAgentEnv.AttachPersistentDisk(apiv1.NewDiskCID("fake-disk-id"), "/fake-guest-persistent-bind-mounts-dir/fake-disk-id")
					Expect(agentEnvService.UpdateAgentEnv).To(Equal(afterAgentEnv))
				})

				Context("when updating agent env succeeds", func() {
					It("returns without an error", func() {
						err := vm.AttachDisk(disk)
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("when updating agent env fails", func() {
					It("returns error", func() {
						agentEnvService.UpdateErr = errors.New("fake-update-err")

						err := vm.AttachDisk(disk)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-update-err"))
					})
				})
			})

			Context("when mounting persistent bind mounts dir fails", func() {
				It("returns error", func() {
					hostBindMounts.MountPersistentErr = errors.New("fake-mount-err")

					err := vm.AttachDisk(disk)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-mount-err"))
				})
			})
		})

		Context("when fetching agent env fails", func() {
			It("returns error", func() {
				agentEnvService.FetchErr = errors.New("fake-fetch-err")

				err := vm.AttachDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-fetch-err"))
			})
		})
	})

	Describe("DetachDisk", func() {
		var (
			disk *fakedisk.FakeDisk
		)

		BeforeEach(func() {
			agentEnv := &apiv1.AgentEnvImpl{}
			agentEnvService.FetchAgentEnv = agentEnv

			disk = fakedisk.NewFakeDisk(apiv1.NewDiskCID("fake-disk-id"))
		})

		It("tries to fetch agent env", func() {
			err := vm.DetachDisk(disk)
			Expect(err).ToNot(HaveOccurred())

			Expect(agentEnvService.FetchCalled).To(BeTrue())
		})

		Context("when fetching agent env succeeds", func() {
			BeforeEach(func() {
				agentEnv := &apiv1.AgentEnvImpl{}
				agentEnv.AttachPersistentDisk(apiv1.NewDiskCID("fake-disk-id"), "/fake-hint-path")
				agentEnv.AttachPersistentDisk(apiv1.NewDiskCID("fake-disk-id2"), "/fake-hint-path2")
				agentEnvService.FetchAgentEnv = agentEnv
			})

			It("unmounts persistent bind mounts dir", func() {
				err := vm.DetachDisk(disk)
				Expect(err).ToNot(HaveOccurred())

				Expect(hostBindMounts.UnmountPersistentID).To(Equal(apiv1.NewVMCID("fake-vm-id")))
				Expect(hostBindMounts.UnmountPersistentDiskID).To(Equal(apiv1.NewDiskCID("fake-disk-id")))
			})

			Context("when unmounting persistent bind mounts dir succeeds", func() {
				It("updates agent env detaching persistent disk", func() {
					err := vm.DetachDisk(disk)
					Expect(err).ToNot(HaveOccurred())

					afterAgentEnv := &apiv1.AgentEnvImpl{}
					afterAgentEnv.AttachPersistentDisk(apiv1.NewDiskCID("fake-disk-id2"), "/fake-hint-path2")
					Expect(agentEnvService.UpdateAgentEnv).To(Equal(afterAgentEnv))
				})

				Context("when updating agent env succeeds", func() {
					It("returns without an error", func() {
						err := vm.DetachDisk(disk)
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("when updating agent env fails", func() {
					It("returns error", func() {
						agentEnvService.UpdateErr = errors.New("fake-update-err")

						err := vm.DetachDisk(disk)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-update-err"))
					})
				})
			})

			Context("when unmounting persistent bind mounts dir fails", func() {
				It("returns error", func() {
					hostBindMounts.UnmountPersistentErr = errors.New("fake-unmount-err")

					err := vm.DetachDisk(disk)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-unmount-err"))
				})
			})
		})

		Context("when fetching agent env fails", func() {
			It("returns error", func() {
				agentEnvService.FetchErr = errors.New("fake-fetch-err")

				err := vm.DetachDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-fetch-err"))
			})
		})
	})
})
