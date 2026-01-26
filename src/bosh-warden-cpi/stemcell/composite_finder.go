package stemcell

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type CompositeFinder struct {
	dirPath string
	fs      boshsys.FileSystem
	logger  boshlog.Logger
}

func NewCompositeFinder(dirPath string, fs boshsys.FileSystem, logger boshlog.Logger) CompositeFinder {
	return CompositeFinder{
		dirPath: dirPath,
		fs:      fs,
		logger:  logger,
	}
}

func (f CompositeFinder) Find(id apiv1.StemcellCID) (Stemcell, bool, error) {
	stemcellDir := filepath.Join(f.dirPath, id.AsString())

	if f.fs.FileExists(stemcellDir) {
		lightMetadataFile := filepath.Join(stemcellDir, "light-stemcell.json")
		if f.fs.FileExists(lightMetadataFile) {
			return f.findLightStemcell(id, lightMetadataFile)
		}
		return NewFSStemcell(id, stemcellDir, f.fs, f.logger), true, nil
	}

	if f.isLightStemcellCID(id.AsString()) {
		return NewLightStemcell(id, id.AsString(), f.logger), true, nil
	}

	return nil, false, nil
}

func (f CompositeFinder) findLightStemcell(id apiv1.StemcellCID, metadataFile string) (Stemcell, bool, error) {
	metadataBytes, err := f.fs.ReadFile(metadataFile)
	if err != nil {
		return nil, false, bosherr.WrapError(err, "Reading light stemcell metadata")
	}

	var metadata lightStemcellMetadata
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return nil, false, bosherr.WrapError(err, "Unmarshaling light stemcell metadata")
	}

	if metadata.ImageReference == "" {
		return nil, false, bosherr.Error("Light stemcell metadata missing image_reference")
	}

	return NewLightStemcell(id, metadata.ImageReference, f.logger), true, nil
}

func (f CompositeFinder) isLightStemcellCID(cid string) bool {
	return len(cid) > 0 && (cid[0] != '/' && (len(cid) > 3 && cid[:3] != "../") && (strings.Contains(cid, ":") || strings.Contains(cid, "/")))
}
