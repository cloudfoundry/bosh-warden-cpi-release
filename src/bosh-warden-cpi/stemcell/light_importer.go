package stemcell

import (
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type LightImporter struct {
	fs             boshsys.FileSystem
	metadataParser MetadataParser

	logTag string
	logger boshlog.Logger
}

func NewLightImporter(
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) LightImporter {
	return LightImporter{
		fs:             fs,
		metadataParser: NewMetadataParser(fs),

		logTag: "LightImporter",
		logger: logger,
	}
}

func (i LightImporter) ImportFromPath(imagePath string) (Stemcell, error) {
	i.logger.Debug(i.logTag, "Importing light stemcell from path '%s'", imagePath)

	metadata, isLight, err := i.metadataParser.ParseFromPath(imagePath)
	if err != nil {
		return nil, bosherr.WrapError(err, "Parsing stemcell metadata")
	}

	if !isLight {
		return nil, bosherr.Error("Not a light stemcell")
	}

	imageReference := metadata.GetImageReference()
	if imageReference == "" {
		return nil, bosherr.Error("Light stemcell metadata missing image_reference")
	}

	err = i.validateImageReference(imageReference)
	if err != nil {
		return nil, bosherr.WrapError(err, "Validating image reference")
	}

	i.logger.Debug(i.logTag, "Light stemcell references image: %s", imageReference)

	cid := "light://" + imageReference

	i.logger.Debug(i.logTag, "Imported light stemcell with CID: %s", cid)

	return NewLightStemcell(apiv1.NewStemcellCID(cid), imageReference, i.logger), nil
}

func (i LightImporter) validateImageReference(imageRef string) error {
	if imageRef == "" {
		return bosherr.Error("Image reference cannot be empty")
	}

	if strings.Contains(imageRef, "..") {
		return bosherr.Error("Image reference contains invalid path traversal pattern")
	}

	if strings.HasPrefix(imageRef, "/") || strings.HasPrefix(imageRef, ":") || strings.HasPrefix(imageRef, "@") {
		return bosherr.Errorf("Image reference has invalid format: %s", imageRef)
	}

	i.logger.Debug(i.logTag, "Image reference validated: %s", imageRef)
	return nil
}
