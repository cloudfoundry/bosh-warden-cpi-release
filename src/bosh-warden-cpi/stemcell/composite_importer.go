package stemcell

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"

	"bosh-warden-cpi/util"
)

type CompositeImporter struct {
	fsImporter     FSImporter
	lightImporter  LightImporter
	metadataParser MetadataParser

	logTag string
	logger boshlog.Logger
}

func NewCompositeImporter(
	dirPath string,
	fs boshsys.FileSystem,
	uuidGen boshuuid.Generator,
	decompressor util.Decompressor,
	logger boshlog.Logger,
) CompositeImporter {
	return CompositeImporter{
		fsImporter:     NewFSImporter(dirPath, fs, uuidGen, decompressor, logger),
		lightImporter:  NewLightImporter(fs, uuidGen, logger),
		metadataParser: NewMetadataParser(fs),

		logTag: "CompositeImporter",
		logger: logger,
	}
}

func (i CompositeImporter) ImportFromPath(imagePath string) (Stemcell, error) {
	i.logger.Debug(i.logTag, "Detecting stemcell type for '%s'", imagePath)

	_, isLight, err := i.metadataParser.ParseFromPath(imagePath)
	if err != nil {
		return nil, bosherr.WrapError(err, "Parsing stemcell metadata")
	}

	if isLight {
		i.logger.Debug(i.logTag, "Detected light stemcell, using LightImporter")
		return i.lightImporter.ImportFromPath(imagePath)
	}

	i.logger.Debug(i.logTag, "Detected traditional stemcell, using FSImporter")
	return i.fsImporter.ImportFromPath(imagePath)
}
