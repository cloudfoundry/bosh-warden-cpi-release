package vm

import (
	"path/filepath"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

// FSGuestBindMounts represents bind mounts from the perspective of a VM
type FSGuestBindMounts struct {
	// Path at which single ephemeral disk is mounted
	ephemeralBindMountPath string

	// Directory with sub-directories at which 0+ persistent disks are mounted
	persistentBindMountsDir string

	logger boshlog.Logger
}

func NewFSGuestBindMounts(
	ephemeralBindMountPath string,
	persistentBindMountsDir string,
	logger boshlog.Logger,
) FSGuestBindMounts {
	return FSGuestBindMounts{
		ephemeralBindMountPath:  ephemeralBindMountPath,
		persistentBindMountsDir: persistentBindMountsDir,

		logger: logger,
	}
}

func (gbm FSGuestBindMounts) MakeEphemeral() string {
	return gbm.ephemeralBindMountPath
}

func (gbm FSGuestBindMounts) MakePersistent() string {
	return gbm.persistentBindMountsDir
}

func (gbm FSGuestBindMounts) MountPersistent(id apiv1.DiskCID) string {
	return filepath.Join(gbm.persistentBindMountsDir, id.AsString())
}
