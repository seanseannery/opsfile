package internal

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// semverRe matches a valid semver string (e.g. "1.2.3" or "1.2.3-beta.1").
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-[\w.]+)?$`)

func TestVersion_IsNonEmpty(t *testing.T) {
	require.NotEmpty(t, Version, "Version must not be empty")
}

func TestVersion_IsSemver(t *testing.T) {
	assert.Regexp(t, semverRe, Version, "Version %q is not a valid semver string", Version)
}
