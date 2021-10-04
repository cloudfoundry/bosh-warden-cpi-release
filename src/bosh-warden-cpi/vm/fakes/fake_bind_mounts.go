package fakes

import (
	"path/filepath"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

type FakeGuestBindMounts struct {
	EphemeralBindMountPath  string
	PersistentBindMountsDir string
}

func (gbm FakeGuestBindMounts) MakeEphemeral() string {
	return gbm.EphemeralBindMountPath
}

func (gbm FakeGuestBindMounts) MakePersistent() string {
	return gbm.PersistentBindMountsDir
}

func (gbm FakeGuestBindMounts) MountPersistent(diskID apiv1.DiskCID) string {
	return filepath.Join(gbm.PersistentBindMountsDir, diskID.AsString())
}

type FakeHostBindMounts struct {
	MakeEphemeralID   apiv1.VMCID
	MakeEphemeralPath string
	MakeEphemeralErr  error

	DeleteEphemeralCalled bool
	DeleteEphemeralID     apiv1.VMCID
	DeleteEphemeralErr    error

	MakePersistentID   apiv1.VMCID
	MakePersistentPath string
	MakePersistentErr  error

	DeletePersistentCalled bool
	DeletePersistentID     apiv1.VMCID
	DeletePersistentErr    error

	MountPersistentID       apiv1.VMCID
	MountPersistentDiskID   apiv1.DiskCID
	MountPersistentDiskPath string
	MountPersistentErr      error

	UnmountPersistentID     apiv1.VMCID
	UnmountPersistentDiskID apiv1.DiskCID
	UnmountPersistentErr    error
}

func (hbm *FakeHostBindMounts) MakeEphemeral(id apiv1.VMCID) (string, error) {
	hbm.MakeEphemeralID = id
	return hbm.MakeEphemeralPath, hbm.MakeEphemeralErr
}

func (hbm *FakeHostBindMounts) DeleteEphemeral(id apiv1.VMCID) error {
	hbm.DeleteEphemeralCalled = true
	hbm.DeleteEphemeralID = id
	return hbm.DeleteEphemeralErr
}

func (hbm *FakeHostBindMounts) MakePersistent(id apiv1.VMCID) (string, error) {
	hbm.MakePersistentID = id
	return hbm.MakePersistentPath, hbm.MakePersistentErr
}

func (hbm *FakeHostBindMounts) DeletePersistent(id apiv1.VMCID) error {
	hbm.DeletePersistentCalled = true
	hbm.DeletePersistentID = id
	return hbm.DeletePersistentErr
}

func (hbm *FakeHostBindMounts) MountPersistent(id apiv1.VMCID, diskID apiv1.DiskCID, diskPath string) error {
	hbm.MountPersistentID = id
	hbm.MountPersistentDiskID = diskID
	hbm.MountPersistentDiskPath = diskPath
	return hbm.MountPersistentErr
}

func (hbm *FakeHostBindMounts) UnmountPersistent(id apiv1.VMCID, diskID apiv1.DiskCID) error {
	hbm.UnmountPersistentID = id
	hbm.UnmountPersistentDiskID = diskID
	return hbm.UnmountPersistentErr
}
