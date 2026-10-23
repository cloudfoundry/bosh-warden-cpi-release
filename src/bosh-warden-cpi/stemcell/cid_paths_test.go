package stemcell

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

func TestStemcellLookupsCannotDeleteOutsideStemcellDirectory(t *testing.T) {
	for _, composite := range []bool{false, true} {
		dir := t.TempDir()
		outside := filepath.Join(dir, "outside")
		if err := os.Mkdir(outside, 0700); err != nil {
			t.Fatal(err)
		}
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs := boshsys.NewOsFileSystem(logger)
		var finder Finder = NewFSFinder(filepath.Join(dir, "stemcells"), fs, logger)
		if composite {
			finder = NewCompositeFinder(filepath.Join(dir, "stemcells"), fs, logger)
		}
		stemcell, found, err := finder.Find(apiv1.NewStemcellCID("../outside"))
		if err == nil {
			if found {
				if err := stemcell.Delete(); err != nil {
					t.Fatal(err)
				}
			}
			t.Fatal("unsafe stemcell CID accepted")
		}
		if _, err := os.Stat(outside); err != nil {
			t.Fatal("outside fixture changed")
		}
	}
}
