package disk

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type FSDisk struct {
	id   string
	path string

	fs     boshsys.FileSystem
	logger boshlog.Logger
}

func NewFSDisk(
	id string,
	path string,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) FSDisk {
	return FSDisk{id: id, path: path, fs: fs, logger: logger}
}

func (s FSDisk) ID() string { return s.id }

func (s FSDisk) Path() string { return s.path }

func (s FSDisk) Exists() (bool, error) {
	return s.fs.FileExists(s.path), nil
}

func (s FSDisk) Delete() error {
	err := s.fs.RemoveAll(s.path)
	if err != nil {
		return bosherr.WrapErrorf(err, "Deleting disk '%s'", s.path)
	}

	return nil
}
