package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realPath resolves symlinks so tests work on macOS where /var -> /private/var.
func realPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", path, err)
	}
	return resolved
}

func TestGetClosestOpsfilePath_FoundInCwd(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	if err := os.WriteFile(filepath.Join(tmp, "Opsfile"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(tmp)

	got, err := getClosestOpsfilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tmp {
		t.Errorf("got %q, want %q", got, tmp)
	}
}

func TestGetClosestOpsfilePath_FoundInParent(t *testing.T) {
	parent := realPath(t, t.TempDir())
	child := filepath.Join(parent, "subdir")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "Opsfile"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(child)

	got, err := getClosestOpsfilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != parent {
		t.Errorf("got %q, want %q", got, parent)
	}
}

func TestGetClosestOpsfilePath_NotFound(t *testing.T) {
	tmp := realPath(t, t.TempDir())

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(tmp)

	_, err := getClosestOpsfilePath()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "could not find Opsfile") {
		t.Errorf("error %q does not mention Opsfile not found", err.Error())
	}
}

func TestGetClosestOpsfilePath_DirectoryNamedOpsfileSkipped(t *testing.T) {
	parent := realPath(t, t.TempDir())
	child := filepath.Join(parent, "subdir")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a directory named "Opsfile" in child — should be skipped.
	if err := os.Mkdir(filepath.Join(child, "Opsfile"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Place the real Opsfile in parent.
	if err := os.WriteFile(filepath.Join(parent, "Opsfile"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(child)

	got, err := getClosestOpsfilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != parent {
		t.Errorf("got %q, want %q (directory named Opsfile should be skipped)", got, parent)
	}
}
