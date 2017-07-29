package util

import (
	"os"

	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Decompressor interface {
	Decompress(src, dest string) error
}

type TarDecompressor struct {
	compressor boshcmd.Compressor
	fs         boshsys.FileSystem
}

func NewTarDecompressor(fs boshsys.FileSystem, compressor boshcmd.Compressor) TarDecompressor {
	return TarDecompressor{
		compressor: compressor,
		fs:         fs,
	}
}

func (d TarDecompressor) Decompress(src, dest string) error {
	err := d.fs.MkdirAll(dest, os.FileMode(0755))
	if err != nil {
		return err
	}

	return d.compressor.DecompressFileToDir(src, dest, boshcmd.CompressorOptions{SameOwner: true})
}
