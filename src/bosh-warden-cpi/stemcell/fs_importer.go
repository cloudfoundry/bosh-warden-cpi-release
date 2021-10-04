package stemcell

import (
	"os"
	"path/filepath"

	"bosh-warden-cpi/util"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

type FSImporter struct {
	dirPath string

	fs           boshsys.FileSystem
	uuidGen      boshuuid.Generator
	decompressor util.Decompressor

	logTag string
	logger boshlog.Logger
}

func NewFSImporter(
	dirPath string,
	fs boshsys.FileSystem,
	uuidGen boshuuid.Generator,
	decompressor util.Decompressor,
	logger boshlog.Logger,
) FSImporter {
	return FSImporter{
		dirPath: dirPath,

		fs:           fs,
		uuidGen:      uuidGen,
		decompressor: decompressor,

		logTag: "FSImporter",
		logger: logger,
	}
}

func (i FSImporter) ImportFromPath(imagePath string) (Stemcell, error) {
	i.logger.Debug(i.logTag, "Importing stemcell from path '%s'", imagePath)

	id, err := i.uuidGen.Generate()
	if err != nil {
		return nil, bosherr.WrapError(err, "Generating stemcell id")
	}

	err = i.fs.MkdirAll(i.dirPath, os.FileMode(0755))
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Creating directory '%s'", i.dirPath)
	}

	stemcellPath := filepath.Join(i.dirPath, id)

	err = i.decompressor.Decompress(imagePath, stemcellPath)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Unpacking stemcell '%s' to '%s'", imagePath, stemcellPath)
	}

	i.logger.Debug(i.logTag, "Imported stemcell from path '%s'", imagePath)

	return NewFSStemcell(apiv1.NewStemcellCID(id), stemcellPath, i.fs, i.logger), nil
}
