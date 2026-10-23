package action

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	bwcstem "bosh-warden-cpi/stemcell"
)

func TestDeleteStemcellAcceptsLightCIDsAndRejectsUnsafeCIDs(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside")
	if err := os.Mkdir(outside, 0700); err != nil {
		t.Fatal(err)
	}
	logger := boshlog.NewLogger(boshlog.LevelNone)
	method := NewDeleteStemcellMethod(bwcstem.NewCompositeFinder(filepath.Join(dir, "stemcells"), boshsys.NewOsFileSystem(logger), logger))

	for _, cid := range []string{"light://registry.example/stemcell:tag", "light://registry.example/stemcell@sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"} {
		if err := method.DeleteStemcell(apiv1.NewStemcellCID(cid)); err != nil {
			t.Fatalf("deleting light stemcell %q: %v", cid, err)
		}
	}

	if err := method.DeleteStemcell(apiv1.NewStemcellCID("../outside")); err == nil {
		t.Fatal("unsafe stemcell CID accepted")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("outside fixture changed")
	}
}
