package vm

import (
	"testing"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

func TestHostMountPathsRejectUnsafeCIDsBeforeOperations(t *testing.T) {
	for _, value := range []string{"../outside", "nested/../../outside", "/absolute", ".", "..", "wild*card", "line\nbreak", "back\\slash", "null\x00byte"} {
		t.Run(value, func(t *testing.T) {
			fs := fakesys.NewFakeFileSystem()
			runner := fakesys.NewFakeCmdRunner()
			mounts := NewFSHostBindMounts("/ephemeral", "/persistent", nil, fs, runner, boshlog.NewLogger(boshlog.LevelNone))
			id := apiv1.NewVMCID(value)
			disk := apiv1.NewDiskCID(value)
			_, e1 := mounts.MakeEphemeral(id)
			_, e2 := mounts.MakePersistent(id)
			errs := []error{e1, e2, mounts.DeleteEphemeral(id), mounts.DeletePersistent(id), mounts.MountPersistent(id, apiv1.NewDiskCID("disk"), "/disk"), mounts.UnmountPersistent(id, apiv1.NewDiskCID("disk")), mounts.MountPersistent(apiv1.NewVMCID("vm"), disk, "/disk"), mounts.UnmountPersistent(apiv1.NewVMCID("vm"), disk)}
			for _, err := range errs {
				if err == nil {
					t.Fatalf("unsafe CID %q accepted", value)
				}
			}
			if len(runner.RunCommands) != 0 {
				t.Fatal("unsafe CID reached mount command")
			}
		})
	}
}

func TestUnmountMatchesOnlyExactTarget(t *testing.T) {
	for _, tc := range []struct {
		name, output string
		wantUnmount  bool
	}{{"prefix", "/dev/loop0 on /persistent/vm/disk-other type ext4 (rw)", false}, {"source", "/persistent/vm/disk on /unrelated type ext4 (rw)", false}, {"exact", "/dev/loop0 on /persistent/vm/disk type ext4 (rw)", true}} {
		t.Run(tc.name, func(t *testing.T) {
			runner := fakesys.NewFakeCmdRunner()
			runner.AddCmdResult("mount", fakesys.FakeCmdResult{Stdout: tc.output})
			mounts := NewFSHostBindMounts("/ephemeral", "/persistent", nil, fakesys.NewFakeFileSystem(), runner, boshlog.NewLogger(boshlog.LevelNone))
			if err := mounts.UnmountPersistent(apiv1.NewVMCID("vm"), apiv1.NewDiskCID("disk")); err != nil {
				t.Fatal(err)
			}
			want := 1
			if tc.wantUnmount {
				want = 2
			}
			if len(runner.RunCommands) != want {
				t.Fatalf("commands=%v", runner.RunCommands)
			}
		})
	}
}
