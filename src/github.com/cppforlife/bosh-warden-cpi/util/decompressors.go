package util

import (
	"fmt"
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

type GzipDecompressor struct {
	fs        boshsys.FileSystem
	cmdRunner boshsys.CmdRunner
}

func NewGzipDecompressor(fs boshsys.FileSystem, cmdRunner boshsys.CmdRunner) GzipDecompressor {
	return GzipDecompressor{
		fs:        fs,
		cmdRunner: cmdRunner,
	}
}

func (d GzipDecompressor) Decompress(src, dest string) error {
	err := d.fs.CopyFile(src, dest+".gz")
	if err != nil {
		return err
	}

	stdout, stderr, exitStatus, err := d.cmdRunner.RunCommand("gunzip", dest+".gz")
	if err != nil {
		return err
	}
	if exitStatus != 0 {
		return fmt.Errorf("gunzip exited non-zero: exit status %d stdout: %s, stderr: %s", exitStatus, stdout, stderr)
	}

	return nil
}
