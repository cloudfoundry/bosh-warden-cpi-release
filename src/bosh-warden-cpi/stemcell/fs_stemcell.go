package stemcell

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type FSStemcell struct {
	id      apiv1.StemcellCID
	dirPath string

	fs     boshsys.FileSystem
	logger boshlog.Logger
}

func NewFSStemcell(
	id apiv1.StemcellCID,
	dirPath string,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) FSStemcell {
	return FSStemcell{id: id, dirPath: dirPath, fs: fs, logger: logger}
}

func (s FSStemcell) ID() apiv1.StemcellCID { return s.id }

func (s FSStemcell) DirPath() string { return s.dirPath }

func (s FSStemcell) Delete() error {
	s.logger.Debug("FSStemcell", "Deleting stemcell '%s'", s.id)

	err := s.fs.RemoveAll(s.dirPath)
	if err != nil {
		return bosherr.WrapErrorf(err, "Deleting stemcell directory '%s'", s.dirPath)
	}

	return nil
}
