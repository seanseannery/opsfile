package internal

import (
	"errors"
	"os/exec"
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
