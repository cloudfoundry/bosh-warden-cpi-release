package util_test

import (
	"errors"

	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bosh-warden-cpi/util"
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

var _ = Describe("GzipDecompressor", func() {
	var (
		fs           *fakesys.FakeFileSystem
		cmdRunner    *fakesys.FakeCmdRunner
		decompressor util.GzipDecompressor
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		cmdRunner = fakesys.NewFakeCmdRunner()
		decompressor = util.NewGzipDecompressor(fs, cmdRunner)

		err := fs.WriteFileString("src", "content")
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Decompress", func() {
		It("copies the file over to be gunzipped", func() {
			cmdRunner.AddCmdResult("gunzip dst.gz", fakesys.FakeCmdResult{})
			err := decompressor.Decompress("src", "dst")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.CopyFileCallCount).To(Equal(1))
			Expect(fs.FileExists("dst.gz")).To(BeTrue())

			contents, err := fs.ReadFileString("dst.gz")
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(Equal("content"))
		})

		It("unpacks the destination", func() {
			cmdRunner.AddCmdResult("gunzip dst.gz", fakesys.FakeCmdResult{})
			err := decompressor.Decompress("src", "dst")
			Expect(err).ToNot(HaveOccurred())

			cmd := cmdRunner.RunCommands[0]
			Expect(cmd[0]).To(Equal("gunzip"))
			Expect(cmd[1]).To(Equal("dst.gz"))
		})

		It("returns an error when copying fails", func() {
			fs.CopyFileError = errors.New("no copying")

			err := decompressor.Decompress("src", "dst")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no copying"))
		})

		It("returns an error when gunzipping fails", func() {
			cmdRunner.AddCmdResult("gunzip dst.gz", fakesys.FakeCmdResult{Error: errors.New("no gunzipping")})

			err := decompressor.Decompress("src", "dst")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no gunzipping"))
		})

		It("returns an error when gunzipping exits non-zero", func() {
			cmdRunner.AddCmdResult("gunzip dst.gz", fakesys.FakeCmdResult{Stdout: "out", Stderr: "err", ExitStatus: 123})

			err := decompressor.Decompress("src", "dst")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("gunzip exited non-zero: exit status 123 stdout: out, stderr: err"))
		})
	})
})
