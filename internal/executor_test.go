package internal

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// toLines converts plain strings into ResolvedLines with Silent=false.
func toLines(texts ...string) []ResolvedLine {
	lines := make([]ResolvedLine, 0, len(texts))
	for _, t := range texts {
		lines = append(lines, ResolvedLine{Text: t})
	}
	return lines
}

func TestExecute(t *testing.T) {
	cases := []struct {
		name         string
		lines        []ResolvedLine
		wantErr      bool
		wantExitCode int
	}{
		{
			name:  "single successful command",
			lines: toLines("true"),
		},
		{
			name:  "multiple successful commands",
			lines: toLines("true", "true", "true"),
		},
		{
			name:         "single failing command",
			lines:        toLines("false"),
			wantErr:      true,
			wantExitCode: 1,
		},
		{
			name:         "stops on first failure",
			lines:        toLines("false", "true"),
			wantErr:      true,
			wantExitCode: 1,
		},
		{
			name:         "middle command fails",
			lines:        toLines("true", "false", "true"),
			wantErr:      true,
			wantExitCode: 1,
		},
		{
			name:         "exit code is propagated",
			lines:        toLines("exit 42"),
			wantErr:      true,
			wantExitCode: 42,
		},
		{
			name:  "empty lines list",
			lines: []ResolvedLine{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Execute(tc.lines, "/bin/sh", false, io.Discard)
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
	err := Execute(toLines("this-command-does-not-exist-at-all"), "/bin/sh", false, io.Discard)
	require.Error(t, err)
	assert.ErrorContains(t, err, "this-command-does-not-exist-at-all")
}

func TestExecute_InvalidShellPath(t *testing.T) {
	err := Execute(toLines("echo hello"), "/nonexistent/shell/binary", false, io.Discard)
	require.Error(t, err)
}

func TestExecute_CommandWithPipe(t *testing.T) {
	err := Execute(toLines("echo hello | cat"), "/bin/sh", false, io.Discard)
	assert.NoError(t, err)
}

func TestExecute_StderrConnected(t *testing.T) {
	// A command writing to stderr should not cause an error by itself.
	err := Execute(toLines("echo error-output >&2"), "/bin/sh", false, io.Discard)
	assert.NoError(t, err)
}

// --- Echo behavior tests ---

func TestExecute_EchoesNonSilentLine(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: false},
	}, "/bin/sh", false, &buf)
	require.NoError(t, err)
	assert.Equal(t, "true\n", buf.String())
}

func TestExecute_SkipsSilentLine(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: true},
	}, "/bin/sh", false, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_GlobalSilentSuppressesAll(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: false},
		{Text: "true", Silent: true},
	}, "/bin/sh", true, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_MixedSilentAndNonSilent(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: true},
		{Text: "echo hello", Silent: false},
		{Text: "true", Silent: true},
	}, "/bin/sh", false, &buf)
	require.NoError(t, err)
	assert.Equal(t, "echo hello\n", buf.String())
}

func TestExecute_EchoMultipleNonSilentLines(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: false},
		{Text: "echo hi", Silent: false},
	}, "/bin/sh", false, &buf)
	require.NoError(t, err)
	assert.Equal(t, "true\necho hi\n", buf.String())
}

func TestExecute_AtPrefixOnFailingCommand(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", Silent: true},
	}, "/bin/sh", false, &buf)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Empty(t, buf.String())
}

func TestExecute_AtPrefixInvalidShell(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "echo hello", Silent: true},
	}, "/nonexistent/shell/binary", false, &buf)
	require.Error(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_EmptyLinesWithSilent(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{}, "/bin/sh", true, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// --- IgnoreError behavior tests ---

func TestExecute_IgnoreErrorContinues(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true},
		{Text: "true"},
	}, "/bin/sh", false, &buf)
	require.NoError(t, err)
	assert.Equal(t, "false\ntrue\n", buf.String())
}

func TestExecute_IgnoreErrorExitCode42(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "exit 42", IgnoreError: true},
	}, "/bin/sh", false, io.Discard)
	assert.NoError(t, err)
}

func TestExecute_NonDashLineStillFails(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: false},
	}, "/bin/sh", false, io.Discard)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 1, exitErr.ExitCode())
}

func TestExecute_IgnoreErrorInvalidShell(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "echo hi", IgnoreError: true},
	}, "/nonexistent/shell/binary", false, io.Discard)
	require.Error(t, err, "system-level error (shell not found) should not be ignored")
}

func TestExecute_FailAfterIgnored(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true},
		{Text: "false", IgnoreError: false},
	}, "/bin/sh", false, io.Discard)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 1, exitErr.ExitCode())
}

func TestExecute_IgnoreErrorAndSilent(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true, Silent: true},
	}, "/bin/sh", false, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_IgnoreErrorWithGlobalSilent(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true},
	}, "/bin/sh", true, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_AllLinesIgnoreError(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true},
		{Text: "exit 2", IgnoreError: true},
		{Text: "exit 127", IgnoreError: true},
	}, "/bin/sh", false, io.Discard)
	assert.NoError(t, err)
}

func TestExecute_IgnoreErrorEchoStillShows(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true, Silent: false},
	}, "/bin/sh", false, &buf)
	require.NoError(t, err)
	assert.Equal(t, "false\n", buf.String())
}

func TestExecute_IgnoreErrorCommandNotFound(t *testing.T) {
	// Shell returns exit code 127 for command not found — this is an ExitError,
	// so it should be ignored when IgnoreError is true.
	err := Execute([]ResolvedLine{
		{Text: "nonexistent-binary-xyz-12345", IgnoreError: true},
	}, "/bin/sh", false, io.Discard)
	assert.NoError(t, err)
}
