package action_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/action"
	fakedisk "github.com/cppforlife/bosh-warden-cpi/disk/fakes"
)

var _ = Describe("DeleteDisk", func() {
	var (
		diskFactory *fakedisk.FakeFactory
		action      DeleteDisk
	)

	BeforeEach(func() {
		diskFactory = &fakedisk.FakeFactory{}
		action = NewDeleteDisk(diskFactory)
	})

	Describe("Run", func() {
		Context("when disk is found with given disk cid", func() {
			var (
				disk *fakedisk.FakeDisk
			)

			BeforeEach(func() {
				disk = fakedisk.NewFakeDisk("fake-disk-id")
				diskFactory.FindDisk = disk
			})

			It("deletes disk", func() {
				_, err := action.Run("fake-disk-id")
				Expect(err).ToNot(HaveOccurred())

				Expect(disk.DeleteCalled).To(BeTrue())
			})

			It("returns error if deleting disk fails", func() {
				disk.DeleteErr = errors.New("fake-delete-err")

				_, err := action.Run("fake-disk-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-delete-err"))
			})
		})

		Context("when disk finding fails", func() {
			It("does not return error", func() {
				diskFactory.FindErr = errors.New("fake-find-err")

				_, err := action.Run("fake-disk-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-find-err"))
			})
		})
	})
})
