package stemcell

import (
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
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
	cidString := id.AsString()

	if strings.HasPrefix(cidString, "light://") {
		imageReference := strings.TrimPrefix(cidString, "light://")
		return NewLightStemcell(id, imageReference, f.logger), true, nil
	}

	stemcellDir := filepath.Join(f.dirPath, cidString)
	if f.fs.FileExists(stemcellDir) {
		return NewFSStemcell(id, stemcellDir, f.fs, f.logger), true, nil
	}

	return nil, false, nil
}
