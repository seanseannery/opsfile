package internal

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestExecute(t *testing.T) {
	cases := []struct {
		name         string
		lines        []string
		wantErr      bool
		wantExitCode int
	}{
		{
			name:  "single successful command",
			lines: []string{"true"},
		},
		{
			name:  "multiple successful commands",
			lines: []string{"true", "true", "true"},
		},
		{
			name:         "single failing command",
			lines:        []string{"false"},
			wantErr:      true,
			wantExitCode: 1,
		},
		{
			name:         "stops on first failure",
			lines:        []string{"false", "true"},
			wantErr:      true,
			wantExitCode: 1,
		},
		{
			name:         "middle command fails",
			lines:        []string{"true", "false", "true"},
			wantErr:      true,
			wantExitCode: 1,
		},
		{
			name:         "exit code is propagated",
			lines:        []string{"exit 42"},
			wantErr:      true,
			wantExitCode: 42,
		},
		{
			name:  "empty lines list",
			lines: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Execute(tc.lines, "/bin/sh")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
				}
				if exitErr.ExitCode() != tc.wantExitCode {
					t.Errorf("exit code: got %d, want %d", exitErr.ExitCode(), tc.wantExitCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestExecute_ErrorWrapsCommandString(t *testing.T) {
	err := Execute([]string{"this-command-does-not-exist-at-all"}, "/bin/sh")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "this-command-does-not-exist-at-all") {
		t.Errorf("error %q does not contain the command string", err.Error())
	}
}

func TestExecute_InvalidShellPath(t *testing.T) {
	err := Execute([]string{"echo hello"}, "/nonexistent/shell/binary")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecute_CommandWithPipe(t *testing.T) {
	err := Execute([]string{"echo hello | cat"}, "/bin/sh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_StderrConnected(t *testing.T) {
	// A command writing to stderr should not cause an error by itself.
	err := Execute([]string{"echo error-output >&2"}, "/bin/sh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
