package disk

import (
	"path/filepath"
	"strconv"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

type FSCreator struct {
	dirPath string

	fs        boshsys.FileSystem
	uuidGen   boshuuid.Generator
	cmdRunner boshsys.CmdRunner

	logTag string
	logger boshlog.Logger
}

func NewFSCreator(
	dirPath string,
	fs boshsys.FileSystem,
	uuidGen boshuuid.Generator,
	cmdRunner boshsys.CmdRunner,
	logger boshlog.Logger,
) FSCreator {
	return FSCreator{
		dirPath: dirPath,

		fs:        fs,
		uuidGen:   uuidGen,
		cmdRunner: cmdRunner,

		logTag: "FSCreator",
		logger: logger,
	}
}

func (c FSCreator) Create(size int) (Disk, error) {
	c.logger.Debug(c.logTag, "Creating disk of size '%d'", size)

	id, err := c.uuidGen.Generate()
	if err != nil {
		return nil, bosherr.WrapError(err, "Generating disk id")
	}

	diskPath := filepath.Join(c.dirPath, id)

	err = c.fs.WriteFile(diskPath, []byte{})
	if err != nil {
		return nil, bosherr.WrapError(err, "Creating empty disk")
	}

	sizeStr := strconv.Itoa(size) + "MB"

	_, _, _, err = c.cmdRunner.RunCommand("truncate", "-s", sizeStr, diskPath)
	if err != nil {
		c.cleanUpFile(diskPath)
		return nil, bosherr.WrapErrorf(err, "Resizing disk to '%s'", sizeStr)
	}

	_, _, _, err = c.cmdRunner.RunCommand("/sbin/mkfs", "-t", "ext4", "-F", diskPath)
	if err != nil {
		c.cleanUpFile(diskPath)
		return nil, bosherr.WrapErrorf(err, "Building disk filesystem '%s'", diskPath)
	}

	return NewFSDisk(id, diskPath, c.fs, c.logger), nil
}

func (c FSCreator) cleanUpFile(path string) {
	err := c.fs.RemoveAll(path)
	if err != nil {
		c.logger.Error(c.logTag, "Failed deleting file '%s': %s", path, err.Error())
	}
}
