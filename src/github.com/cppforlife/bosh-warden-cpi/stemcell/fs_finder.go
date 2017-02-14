package stemcell

import (
	"path/filepath"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
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
	dirPath := filepath.Join(f.dirPath, id.AsString())

	if f.fs.FileExists(dirPath) {
		return NewFSStemcell(id, dirPath, f.fs, f.logger), true, nil
	}

	return nil, false, nil
}
