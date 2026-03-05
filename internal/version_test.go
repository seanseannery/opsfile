package internal

import (
	"regexp"
	"testing"
)

// semverRe matches a valid semver string (e.g. "1.2.3" or "1.2.3-beta.1").
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-[\w.]+)?$`)

func TestVersion_IsNonEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
}

func TestVersion_IsSemver(t *testing.T) {
	if !semverRe.MatchString(Version) {
		t.Errorf("Version %q is not a valid semver string", Version)
	}
}
