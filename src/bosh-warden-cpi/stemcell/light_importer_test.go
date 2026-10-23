package stemcell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

func TestLightStemcellPreservesManifestDigestThroughLookup(t *testing.T) {
	hash := strings.Repeat("a", 64)
	expected := "sha256:" + hash
	for _, tc := range []struct {
		name, ref, digest string
		wantOK            bool
	}{{"mutable reference with digest", "registry.example/stemcell:tag", expected, true}, {"plain digest", "registry.example/stemcell:tag", hash, true}, {"already pinned", "registry.example/stemcell@" + expected, "", true}, {"tag without digest", "registry.example/stemcell:tag", "", false}, {"digest conflict", "registry.example/stemcell@" + expected, "sha256:" + strings.Repeat("b", 64), false}, {"invalid digest", "registry.example/stemcell:tag", "sha256:abc", false}, {"invalid reference", "../stemcell", expected, false}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			data := fmt.Sprintf("stemcell_formats: [warden-light]\ncloud_properties:\n  image_reference: %q\n  digest: %q\n", tc.ref, tc.digest)
			if err := os.WriteFile(filepath.Join(dir, "stemcell.MF"), []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			logger := boshlog.NewLogger(boshlog.LevelNone)
			fs := boshsys.NewOsFileSystem(logger)
			image, err := NewLightImporter(fs, logger).ImportFromPath(dir)
			if !tc.wantOK {
				if err == nil {
					t.Fatal("accepted unpinned or conflicting image")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			expectedRef := "registry.example/stemcell@" + expected
			if image.ID().AsString() != "light://"+expectedRef {
				t.Fatalf("CID %s discarded manifest digest", image.ID().AsString())
			}
			found, exists, err := NewCompositeFinder(dir, fs, logger).Find(image.ID())
			if err != nil || !exists {
				t.Fatalf("lookup: %v", err)
			}
			if found.URI() != "docker://"+expectedRef {
				t.Fatalf("Garden URI %s is not content-addressed", found.URI())
			}
		})
	}
	logger := boshlog.NewLogger(boshlog.LevelNone)
	_, _, err := NewCompositeFinder(t.TempDir(), boshsys.NewOsFileSystem(logger), logger).Find(apiv1.NewStemcellCID("light://registry.example/stemcell:tag"))
	if err == nil {
		t.Fatal("mutable legacy light CID accepted without metadata digest")
	}
}
