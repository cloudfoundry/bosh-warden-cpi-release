package util

import (
	"fmt"
	"regexp"
)

var cidComponent = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidateCIDPathComponent accepts the filename form of locally minted CIDs.
// Reject separators, dot components, glob patterns and control characters before
// any filesystem or mount operation uses a caller-supplied ID.
func ValidateCIDPathComponent(value string) error {
	if !cidComponent.MatchString(value) {
		return fmt.Errorf("CID must be a single safe path component")
	}
	return nil
}
