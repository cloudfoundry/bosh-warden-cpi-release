package disk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

func TestDiskLookupCannotDeleteOutsideDiskDirectory(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside")
	if err := os.WriteFile(outside, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	logger := boshlog.NewLogger(boshlog.LevelNone)
	factory := NewFSFactory(filepath.Join(dir, "disks"), boshsys.NewOsFileSystem(logger), nil, nil, logger)
	disk, err := factory.Find(apiv1.NewDiskCID("../outside"))
	if err == nil {
		if err := disk.Delete(); err != nil {
			t.Fatal(err)
		}
		t.Fatal("unsafe CID reached delete")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("outside fixture changed")
	}
	disk, err = factory.Find(apiv1.NewDiskCID("ordinary-id"))
	if err != nil || disk.Path() != filepath.Join(dir, "disks", "ordinary-id") {
		t.Fatalf("normal disk lookup: %v", err)
	}
}
