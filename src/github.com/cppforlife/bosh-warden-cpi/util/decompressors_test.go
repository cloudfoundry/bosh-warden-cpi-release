package util_test

import (
	"errors"

	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cppforlife/bosh-warden-cpi/util"
)

var _ = Describe("TarDecompressor", func() {
	var (
		fs           *fakesys.FakeFileSystem
		compressor   *fakecmd.FakeCompressor
		decompressor util.TarDecompressor
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		compressor = fakecmd.NewFakeCompressor()
		decompressor = util.NewTarDecompressor(fs, compressor)
	})

	Describe("Decompress", func() {
		It("creates directory in collection directory that will contain unpacked stemcell", func() {
			err := decompressor.Decompress("src", "dst")
			Expect(err).ToNot(HaveOccurred())

			unpackDirStat := fs.GetFileTestStat("dst")
			Expect(unpackDirStat.FileType).To(Equal(fakesys.FakeFileTypeDir))
			Expect(int(unpackDirStat.FileMode)).To(Equal(0755)) // todo
		})

		It("unpacks the src argument correctly", func() {
			err := decompressor.Decompress("src", "dst")
			Expect(err).ToNot(HaveOccurred())

			Expect(compressor.DecompressFileToDirTarballPaths[0]).To(Equal("src"))
			Expect(compressor.DecompressFileToDirDirs[0]).To(Equal("dst"))
			Expect(compressor.DecompressFileToDirOptions[0]).To(Equal(boshcmd.CompressorOptions{SameOwner: true}))
		})

		It("returns error if creating directory that will contain unpacked stemcell fails", func() {
			fs.MkdirAllError = errors.New("fake-mkdir-all-err")

			err := decompressor.Decompress("src", "dst")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-err"))
		})

		It("returns error if unpacking fails", func() {
			compressor.DecompressFileToDirErr = errors.New("fake-decompress-err")

			err := decompressor.Decompress("src", "dst")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-decompress-err"))
		})
	})
})
