package internal

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				require.Error(t, err)
				var exitErr *exec.ExitError
				require.True(t, errors.As(err, &exitErr), "expected *exec.ExitError, got %T: %v", err, err)
				assert.Equal(t, tc.wantExitCode, exitErr.ExitCode())
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestExecute_ErrorWrapsCommandString(t *testing.T) {
	err := Execute([]string{"this-command-does-not-exist-at-all"}, "/bin/sh")
	require.Error(t, err)
	assert.ErrorContains(t, err, "this-command-does-not-exist-at-all")
}

func TestExecute_InvalidShellPath(t *testing.T) {
	err := Execute([]string{"echo hello"}, "/nonexistent/shell/binary")
	require.Error(t, err)
}

func TestExecute_CommandWithPipe(t *testing.T) {
	err := Execute([]string{"echo hello | cat"}, "/bin/sh")
	assert.NoError(t, err)
}

func TestExecute_StderrConnected(t *testing.T) {
	// A command writing to stderr should not cause an error by itself.
	err := Execute([]string{"echo error-output >&2"}, "/bin/sh")
	assert.NoError(t, err)
}
