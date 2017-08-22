package stemcell_test

import (
	"errors"
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/stemcell"
)

var _ = Describe("FSImporter", func() {
	var (
		fs       *fakesys.FakeFileSystem
		stemcell FSStemcell
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		stemcell = NewFSStemcell(
			apiv1.NewStemcellCID("fake-stemcell-id"), "/fake-stemcell-dir", fs, logger)
	})

	Describe("Delete", func() {
		It("deletes directory in collection directory that contains unpacked stemcell", func() {
			err := fs.MkdirAll("/fake-stemcell-dir", os.ModeDir)
			Expect(err).ToNot(HaveOccurred())

			err = stemcell.Delete()
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("/fake-stemcell-dir")).To(BeFalse())
		})

		It("returns error if deleting stemcell directory fails", func() {
			fs.RemoveAllStub = func(string) error {
				return errors.New("fake-remove-all-err")
			}

			err := stemcell.Delete()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-remove-all-err"))
		})
	})
})
