package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realPath resolves symlinks so tests work on macOS where /var -> /private/var.
func realPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err, "EvalSymlinks(%q)", path)
	return resolved
}

func TestGetClosestOpsfilePath_FoundInCwd(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	err := os.WriteFile(filepath.Join(tmp, "Opsfile"), []byte(""), 0o644)
	require.NoError(t, err)

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(tmp)

	got, err := getClosestOpsfilePath()
	require.NoError(t, err)
	assert.Equal(t, tmp, got)
}

func TestGetClosestOpsfilePath_FoundInParent(t *testing.T) {
	parent := realPath(t, t.TempDir())
	child := filepath.Join(parent, "subdir")
	require.NoError(t, os.Mkdir(child, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(parent, "Opsfile"), []byte(""), 0o644))

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(child)

	got, err := getClosestOpsfilePath()
	require.NoError(t, err)
	assert.Equal(t, parent, got)
}

func TestGetClosestOpsfilePath_NotFound(t *testing.T) {
	tmp := realPath(t, t.TempDir())

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(tmp)

	_, err := getClosestOpsfilePath()
	require.Error(t, err)
	assert.ErrorContains(t, err, "could not find Opsfile")
}

func TestGetClosestOpsfilePath_DirectoryNamedOpsfileSkipped(t *testing.T) {
	parent := realPath(t, t.TempDir())
	child := filepath.Join(parent, "subdir")
	require.NoError(t, os.Mkdir(child, 0o755))
	// Create a directory named "Opsfile" in child — should be skipped.
	require.NoError(t, os.Mkdir(filepath.Join(child, "Opsfile"), 0o755))
	// Place the real Opsfile in parent.
	require.NoError(t, os.WriteFile(filepath.Join(parent, "Opsfile"), []byte(""), 0o644))

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(child)

	got, err := getClosestOpsfilePath()
	require.NoError(t, err)
	assert.Equal(t, parent, got, "directory named Opsfile should be skipped")
}
