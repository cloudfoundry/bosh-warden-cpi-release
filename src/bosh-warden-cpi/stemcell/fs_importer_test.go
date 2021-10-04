package stemcell_test

import (
	"errors"
	"os"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "bosh-warden-cpi/stemcell"
	"bosh-warden-cpi/stemcell/fakes"
)

var _ = Describe("FSImporter", func() {
	var (
		fs           *fakesys.FakeFileSystem
		uuidGen      *fakeuuid.FakeGenerator
		decompressor *fakes.FakeDecompressor
		logger       boshlog.Logger
		importer     FSImporter
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		uuidGen = &fakeuuid.FakeGenerator{}
		decompressor = &fakes.FakeDecompressor{}
		logger = boshlog.NewLogger(boshlog.LevelNone)
		importer = NewFSImporter("/fake-collection-dir", fs, uuidGen, decompressor, logger)
	})

	Describe("ImportFromPath", func() {
		It("makes the directory in which to unpack the stemcell", func() {
			uuidGen.GeneratedUUID = "fake-uuid"

			_, err := importer.ImportFromPath("/fake-image-path")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("/fake-collection-dir")).To(BeTrue())

			stat, err := fs.Stat("/fake-collection-dir")
			Expect(err).NotTo(HaveOccurred())
			Expect(stat.Mode()).To(Equal(os.FileMode(0755)))
		})

		It("returns unique stemcell id", func() {
			uuidGen.GeneratedUUID = "fake-uuid"

			stemcell, err := importer.ImportFromPath("/fake-image-path")
			Expect(err).ToNot(HaveOccurred())

			expectedStemcell := NewFSStemcell(
				apiv1.NewStemcellCID("fake-uuid"), "/fake-collection-dir/fake-uuid", fs, logger)
			Expect(stemcell).To(Equal(expectedStemcell))
		})

		It("returns error if generating stemcell id fails", func() {
			uuidGen.GenerateError = errors.New("fake-generate-err")

			stemcell, err := importer.ImportFromPath("/fake-image-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-generate-err"))
			Expect(stemcell).To(BeNil())
		})

		It("unpacks stemcell into directory that will contain this unpacked stemcell", func() {
			uuidGen.GeneratedUUID = "fake-uuid"

			_, err := importer.ImportFromPath("/fake-image-path")
			Expect(err).ToNot(HaveOccurred())

			Expect(decompressor.DecompressSrcForCall[0]).To(Equal("/fake-image-path"))
			Expect(decompressor.DecompressDstForCall[0]).To(Equal("/fake-collection-dir/fake-uuid"))
		})

		It("returns error if creating directory fails", func() {
			fs.MkdirAllError = errors.New("fake-mkdir-all-error")

			stemcell, err := importer.ImportFromPath("/fake-image-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-error"))
			Expect(stemcell).To(BeNil())
		})

		It("returns error if unpacking stemcell fails", func() {
			decompressor.DecompressError = errors.New("fake-decompress-error")

			stemcell, err := importer.ImportFromPath("/fake-image-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-decompress-err"))
			Expect(stemcell).To(BeNil())
		})
	})
})
