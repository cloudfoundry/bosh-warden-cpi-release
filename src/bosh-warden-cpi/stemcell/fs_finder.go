package stemcell

import (
	bwcutil "bosh-warden-cpi/util"
	"path/filepath"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type FSFinder struct {
	dirPath string

	fs     boshsys.FileSystem
	logger boshlog.Logger
}

func NewFSFinder(dirPath string, fs boshsys.FileSystem, logger boshlog.Logger) FSFinder {
	return FSFinder{dirPath: dirPath, fs: fs, logger: logger}
}

func (f FSFinder) Find(id apiv1.StemcellCID) (Stemcell, bool, error) {
	if err := bwcutil.ValidateCIDPathComponent(id.AsString()); err != nil {
		return nil, false, err
	}
	dirPath := filepath.Join(f.dirPath, id.AsString())

	if f.fs.FileExists(dirPath) {
		return NewFSStemcell(id, dirPath, f.fs, f.logger), true, nil
	}

	return nil, false, nil
}
