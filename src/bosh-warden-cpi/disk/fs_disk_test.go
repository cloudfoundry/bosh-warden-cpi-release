package disk_test

import (
	"errors"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "bosh-warden-cpi/disk"
)

var _ = Describe("FSDisk", func() {
	var (
		fs   *fakesys.FakeFileSystem
		disk FSDisk
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		disk = NewFSDisk(apiv1.NewDiskCID("fake-disk-id"), "/fake-disk-path", fs, logger)
	})

	Describe("Delete", func() {
		It("deletes path", func() {
			err := fs.WriteFileString("/fake-disk-path", "fake-content")
			Expect(err).ToNot(HaveOccurred())

			err = disk.Delete()
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("/fake-disk-path")).To(BeFalse())
		})

		It("returns error if deleting path fails", func() {
			fs.RemoveAllStub = func(string) error {
				return errors.New("fake-remove-all-err")
			}

			err := disk.Delete()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-remove-all-err"))
		})
	})
})
