package vm

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	bwcutil "bosh-warden-cpi/util"
)

// FSHostBindMounts represents bind mounts from the perspective of the host
type FSHostBindMounts struct {
	// Directory with sub-directories at which ephemeral disks are mounted
	ephemeralBindMountsDir string

	// Directory with sub-directories at which ephemeral disks are mounted
	persistentBindMountsDir string

	sleeper   bwcutil.Sleeper
	fs        boshsys.FileSystem
	cmdRunner boshsys.CmdRunner
	logger    boshlog.Logger
}

func NewFSHostBindMounts(
	ephemeralBindMountsDir string,
	persistentBindMountsDir string,
	sleeper bwcutil.Sleeper,
	fs boshsys.FileSystem,
	cmdRunner boshsys.CmdRunner,
	logger boshlog.Logger,
) FSHostBindMounts {
	return FSHostBindMounts{
		ephemeralBindMountsDir:  ephemeralBindMountsDir,
		persistentBindMountsDir: persistentBindMountsDir,

		sleeper:   sleeper,
		fs:        fs,
		cmdRunner: cmdRunner,
		logger:    logger,
	}
}

func (hbm FSHostBindMounts) MakeEphemeral(id apiv1.VMCID) (string, error) {
	path := filepath.Join(hbm.ephemeralBindMountsDir, id.AsString())

	err := hbm.fs.MkdirAll(path, os.FileMode(0755))
	if err != nil {
		return "", bosherr.WrapError(err, "Making ephemeral bind mount")
	}

	// --make-shared keeps this mount in the same peer group as the host-side
	// copy so that container-internal mounts (e.g. systemd's /run tmpfs) that
	// propagate to the host also propagate back into BPM's namespace, making
	// them visible to umount --recursive in DeleteEphemeral.
	mountArgss := [][]string{
		[]string{"--bind", path, path},
		[]string{"--make-shared", path},
	}

	for _, mountArgs := range mountArgss {
		_, _, _, err = hbm.cmdRunner.RunCommand("mount", mountArgs...)
		if err != nil {
			return "", err
		}
	}

	return path, nil
}

func (hbm FSHostBindMounts) DeleteEphemeral(id apiv1.VMCID) error {
	path := filepath.Join(hbm.ephemeralBindMountsDir, id.AsString())

	if !hbm.fs.FileExists(path) {
		return nil
	}

	// With shared: true on the BPM unrestricted_volume for /var/vcap/store/warden_cpi,
	// the mount --bind in MakeEphemeral propagates to the host namespace as a shared
	// mount. Garden then binds that host-side path into the VM container as
	// /var/vcap/data. Any mounts made inside the container (e.g. a systemd tmpfs at
	// /run, visible as /var/vcap/data/sys/run) propagate back to the host through the
	// shared mount, appearing as nested mounts under this path. A plain umount only
	// removes the top-level self-bind mount and leaves nested mounts in place,
	// causing the subsequent RemoveAll to fail with "device or resource busy".
	// umount --recursive tears down the entire mount tree before we delete.
	_, _, _, err := hbm.cmdRunner.RunCommand("umount", "--recursive", path)
	if err != nil && !strings.Contains(err.Error(), "not mounted") {
		return err
	}

	err = hbm.deletePath(path)
	if err != nil {
		return bosherr.WrapError(err, "Removing ephemeral bind mount")
	}

	return nil
}

func (hbm FSHostBindMounts) MakePersistent(id apiv1.VMCID) (string, error) {
	path := filepath.Join(hbm.persistentBindMountsDir, id.AsString())

	err := hbm.fs.MkdirAll(path, os.FileMode(0755))
	if err != nil {
		return "", bosherr.WrapError(err, "Making persistent bind mounts")
	}

	mountArgss := [][]string{
		[]string{"--bind", path, path},

		// An unbindable mount is a private mount which cannot be cloned through a bind operation.
		[]string{"--make-unbindable", path},

		// A shared mount provides ability to create mirrors of that mount such that mounts and
		// umounts within any of the mirrors propagate to the other mirror.
		[]string{"--make-shared", path},
	}

	for _, mountArgs := range mountArgss {
		_, _, _, err = hbm.cmdRunner.RunCommand("mount", mountArgs...)
		if err != nil {
			return "", err
		}
	}

	return path, nil
}

func (hbm FSHostBindMounts) DeletePersistent(id apiv1.VMCID) error {
	path := filepath.Join(hbm.persistentBindMountsDir, id.AsString())

	if hbm.fs.FileExists(path) {
		mountedDiskPaths, err := hbm.fs.Glob(filepath.Join(path, "*"))
		if err != nil {
			return bosherr.WrapErrorf(err, "Getting mounted disk paths in '%s'", path)
		}

		for _, mountedDiskPath := range mountedDiskPaths {
			err := hbm.unmountPath(mountedDiskPath)
			if err != nil {
				return bosherr.WrapErrorf(err, "Unmounting persistent disk '%s'", mountedDiskPath)
			}
		}

		_, _, _, err = hbm.cmdRunner.RunCommand("umount", path)
		if err != nil && !strings.Contains(err.Error(), "not mounted") {
			return err
		}

		err = hbm.deletePath(path)
		if err != nil {
			return bosherr.WrapError(err, "Removing persistent bind mounts")
		}
	}

	return nil
}

func (hbm FSHostBindMounts) MountPersistent(id apiv1.VMCID, diskID apiv1.DiskCID, diskPath string) error {
	path := filepath.Join(hbm.persistentBindMountsDir, id.AsString(), diskID.AsString())

	err := hbm.fs.MkdirAll(path, os.FileMode(0755))
	if err != nil {
		return bosherr.WrapError(err, "Making disk specific persistent bind mount")
	}

	_, _, _, err = hbm.cmdRunner.RunCommand("mount", diskPath, path, "-o", "loop")
	if err != nil {
		return bosherr.WrapError(err, "Mounting disk specific persistent bind mount")
	}

	return nil
}

func (hbm FSHostBindMounts) UnmountPersistent(id apiv1.VMCID, diskID apiv1.DiskCID) error {
	path := filepath.Join(hbm.persistentBindMountsDir, id.AsString(), diskID.AsString())
	return hbm.unmountPath(path)
}

func (hbm FSHostBindMounts) unmountPath(path string) error {
	var lastErr error

	for i := 0; i < 60; i++ {
		// Check for all mounts on the host
		stdout, _, _, err := hbm.cmdRunner.RunCommand("mount")
		if err != nil {
			return bosherr.WrapError(err, "Checking persistent bind mount")
		}

		// If output does not contain path it means that either
		// it was never mounted or it was successfully unmounted
		if !strings.Contains(stdout, path) {
			return nil
		}

		// Try unmounting again; otherwise, try doing it later
		_, _, _, lastErr = hbm.cmdRunner.RunCommand("umount", path)
		if lastErr == nil {
			return nil
		}

		hbm.sleeper.Sleep(3 * time.Second)
	}

	return bosherr.WrapError(lastErr, "Unmounting disk specific persistent bind mount")
}

func (hbm FSHostBindMounts) deletePath(path string) error {
	var lastErr error

	// Try multiple times to avoid 'device or resource busy' error
	for i := 0; i < 60; i++ {
		lastErr = hbm.fs.RemoveAll(path)
		if lastErr == nil {
			return nil
		}

		hbm.sleeper.Sleep(500 * time.Millisecond)
	}

	return lastErr
}
