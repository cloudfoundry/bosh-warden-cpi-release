package action_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/action"
	fakedisk "github.com/cppforlife/bosh-warden-cpi/disk/fakes"
	fakevm "github.com/cppforlife/bosh-warden-cpi/vm/fakes"
)

var _ = Describe("DetachDisk", func() {
	var (
		vmFinder    *fakevm.FakeFinder
		diskFactory *fakedisk.FakeFactory
		action      DetachDisk
	)

	BeforeEach(func() {
		vmFinder = &fakevm.FakeFinder{}
		diskFactory = &fakedisk.FakeFactory{}
		action = NewDetachDisk(vmFinder, diskFactory)
	})

	Describe("Run", func() {
		It("tries to find VM with given VM cid", func() {
			vmFinder.FindFound = true
			vmFinder.FindVM = fakevm.NewFakeVM("fake-vm-id")

			diskFactory.FindDisk = fakedisk.NewFakeDisk("fake-disk-id")

			_, err := action.Run("fake-vm-id", "fake-disk-id")
			Expect(err).ToNot(HaveOccurred())

			Expect(vmFinder.FindID).To(Equal("fake-vm-id"))
		})

		Context("when VM is found with given VM cid", func() {
			var (
				vm *fakevm.FakeVM
			)

			BeforeEach(func() {
				vm = fakevm.NewFakeVM("fake-vm-id")
				vmFinder.FindVM = vm
				vmFinder.FindFound = true
			})

			It("tries to find disk with given disk cid", func() {
				_, err := action.Run("fake-vm-id", "fake-disk-id")
				Expect(err).ToNot(HaveOccurred())

				Expect(diskFactory.FindID).To(Equal("fake-disk-id"))
			})

			Context("when disk is found with given disk cid", func() {
				var (
					disk *fakedisk.FakeDisk
				)

				BeforeEach(func() {
					disk = fakedisk.NewFakeDisk("fake-disk-id")
					diskFactory.FindDisk = disk
				})

				It("does not return error when detaching found disk from found VM succeeds", func() {
					_, err := action.Run("fake-vm-id", "fake-disk-id")
					Expect(err).ToNot(HaveOccurred())

					Expect(vm.DetachDiskDisk).To(Equal(disk))
				})

				It("returns error if detaching disk fails", func() {
					vm.DetachDiskErr = errors.New("fake-detach-disk-err")

					_, err := action.Run("fake-vm-id", "fake-disk-id")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-detach-disk-err"))
				})
			})

			Context("when disk finding fails", func() {
				It("returns error", func() {
					diskFactory.FindErr = errors.New("fake-find-err")

					_, err := action.Run("fake-vm-id", "fake-disk-id")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-find-err"))
				})
			})
		})

		Context("when VM is not found with given cid", func() {
			It("returns error because disk can only be detached from an existing VM", func() {
				vmFinder.FindFound = false

				_, err := action.Run("fake-vm-id", "fake-disk-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Expected to find VM"))
			})
		})

		Context("when VM finding fails", func() {
			It("returns error because disk can only be detached from an existing VM", func() {
				vmFinder.FindErr = errors.New("fake-find-err")

				_, err := action.Run("fake-vm-id", "fake-disk-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-find-err"))
			})
		})
	})
})
