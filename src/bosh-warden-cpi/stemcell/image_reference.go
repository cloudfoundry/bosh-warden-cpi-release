package stemcell

import (
	_ "crypto/sha256" // Register SHA256 for OCI digest validation.
	"fmt"
	"strings"

	"github.com/distribution/reference"
	digest "github.com/opencontainers/go-digest"
)

// Preserve the manifest's expected content digest all the way to Garden's
// content-addressed image pull. Mutable tag-only CIDs are rejected on lookup.
func pinnedLightImageReference(imageRef, expected string) (string, error) {
	named, err := reference.ParseNormalizedNamed(imageRef)
	if err != nil {
		return "", fmt.Errorf("invalid light stemcell image reference: %w", err)
	}
	canonical, alreadyPinned := named.(reference.Canonical)
	if expected == "" {
		if !alreadyPinned {
			return "", fmt.Errorf("light stemcell requires a SHA256 image digest")
		}
		expected = canonical.Digest().String()
	}
	if !strings.HasPrefix(expected, "sha256:") {
		expected = "sha256:" + expected
	}
	expectedDigest, err := digest.Parse(expected)
	if err != nil || expectedDigest.Algorithm() != digest.SHA256 {
		return "", fmt.Errorf("light stemcell requires a valid SHA256 image digest")
	}
	if alreadyPinned && canonical.Digest() != expectedDigest {
		return "", fmt.Errorf("image reference digest does not match stemcell metadata")
	}
	pinned, err := reference.WithDigest(reference.TrimNamed(named), expectedDigest)
	if err != nil {
		return "", fmt.Errorf("pinning light stemcell image reference: %w", err)
	}
	return pinned.String(), nil
}
