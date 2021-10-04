package disk

import (
	"path/filepath"
	"strconv"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

type FSFactory struct {
	dirPath string

	fs        boshsys.FileSystem
	uuidGen   boshuuid.Generator
	cmdRunner boshsys.CmdRunner

	logTag string
	logger boshlog.Logger
}

func NewFSFactory(
	dirPath string,
	fs boshsys.FileSystem,
	uuidGen boshuuid.Generator,
	cmdRunner boshsys.CmdRunner,
	logger boshlog.Logger,
) FSFactory {
	return FSFactory{
		dirPath: dirPath,

		fs:        fs,
		uuidGen:   uuidGen,
		cmdRunner: cmdRunner,

		logTag: "disk.FSFactory",
		logger: logger,
	}
}

func (f FSFactory) Create(size int) (Disk, error) {
	f.logger.Debug(f.logTag, "Creating disk of size '%d'", size)

	id, err := f.uuidGen.Generate()
	if err != nil {
		return nil, bosherr.WrapError(err, "Generating disk id")
	}

	diskPath := filepath.Join(f.dirPath, id)

	err = f.fs.WriteFile(diskPath, []byte{})
	if err != nil {
		return nil, bosherr.WrapError(err, "Creating empty disk")
	}

	sizeStr := strconv.Itoa(size) + "MB"

	_, _, _, err = f.cmdRunner.RunCommand("truncate", "-s", sizeStr, diskPath)
	if err != nil {
		f.cleanUpFile(diskPath)
		return nil, bosherr.WrapErrorf(err, "Resizing disk to '%s'", sizeStr)
	}

	_, _, _, err = f.cmdRunner.RunCommand("/sbin/mkfs", "-t", "ext4", "-F", diskPath)
	if err != nil {
		f.cleanUpFile(diskPath)
		return nil, bosherr.WrapErrorf(err, "Building disk filesystem '%s'", diskPath)
	}

	return NewFSDisk(apiv1.NewDiskCID(id), diskPath, f.fs, f.logger), nil
}

func (f FSFactory) Find(id apiv1.DiskCID) (Disk, error) {
	return NewFSDisk(id, filepath.Join(f.dirPath, id.AsString()), f.fs, f.logger), nil
}

func (f FSFactory) cleanUpFile(path string) {
	err := f.fs.RemoveAll(path)
	if err != nil {
		f.logger.Error(f.logTag, "Failed deleting file '%s': %s", path, err.Error())
	}
}
