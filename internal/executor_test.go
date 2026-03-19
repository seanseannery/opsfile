package internal

import (
	"bytes"
	"errors"
	"io"
	"os"
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
			err := Execute(tc.lines, "/bin/sh", false, false, nil, io.Discard)
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
	err := Execute(toLines("this-command-does-not-exist-at-all"), "/bin/sh", false, false, nil, io.Discard)
	require.Error(t, err)
	assert.ErrorContains(t, err, "this-command-does-not-exist-at-all")
}

func TestExecute_InvalidShellPath(t *testing.T) {
	err := Execute(toLines("echo hello"), "/nonexistent/shell/binary", false, false, nil, io.Discard)
	require.Error(t, err)
}

func TestExecute_CommandWithPipe(t *testing.T) {
	err := Execute(toLines("echo hello | cat"), "/bin/sh", false, false, nil, io.Discard)
	assert.NoError(t, err)
}

func TestExecute_StderrConnected(t *testing.T) {
	// A command writing to stderr should not cause an error by itself.
	err := Execute(toLines("echo error-output >&2"), "/bin/sh", false, false, nil, io.Discard)
	assert.NoError(t, err)
}

// --- Echo behavior tests ---

func TestExecute_EchoesNonSilentLine(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: false},
	}, "/bin/sh", false, false, nil, &buf)
	require.NoError(t, err)
	assert.Equal(t, "true\n", buf.String())
}

func TestExecute_SkipsSilentLine(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: true},
	}, "/bin/sh", false, false, nil, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_GlobalSilentSuppressesAll(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: false},
		{Text: "true", Silent: true},
	}, "/bin/sh", true, false, nil, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_MixedSilentAndNonSilent(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: true},
		{Text: "echo hello", Silent: false},
		{Text: "true", Silent: true},
	}, "/bin/sh", false, false, nil, &buf)
	require.NoError(t, err)
	assert.Equal(t, "echo hello\n", buf.String())
}

func TestExecute_EchoMultipleNonSilentLines(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "true", Silent: false},
		{Text: "echo hi", Silent: false},
	}, "/bin/sh", false, false, nil, &buf)
	require.NoError(t, err)
	assert.Equal(t, "true\necho hi\n", buf.String())
}

func TestExecute_AtPrefixOnFailingCommand(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", Silent: true},
	}, "/bin/sh", false, false, nil, &buf)
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
	}, "/nonexistent/shell/binary", false, false, nil, &buf)
	require.Error(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_EmptyLinesWithSilent(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{}, "/bin/sh", true, false, nil, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// --- IgnoreError behavior tests ---

func TestExecute_IgnoreErrorContinues(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true},
		{Text: "true"},
	}, "/bin/sh", false, false, nil, &buf)
	require.NoError(t, err)
	assert.Equal(t, "false\ntrue\n", buf.String())
}

func TestExecute_IgnoreErrorExitCode42(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "exit 42", IgnoreError: true},
	}, "/bin/sh", false, false, nil, io.Discard)
	assert.NoError(t, err)
}

func TestExecute_NonDashLineStillFails(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: false},
	}, "/bin/sh", false, false, nil, io.Discard)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 1, exitErr.ExitCode())
}

func TestExecute_IgnoreErrorInvalidShell(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "echo hi", IgnoreError: true},
	}, "/nonexistent/shell/binary", false, false, nil, io.Discard)
	require.Error(t, err, "system-level error (shell not found) should not be ignored")
}

func TestExecute_FailAfterIgnored(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true},
		{Text: "false", IgnoreError: false},
	}, "/bin/sh", false, false, nil, io.Discard)
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 1, exitErr.ExitCode())
}

func TestExecute_IgnoreErrorAndSilent(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true, Silent: true},
	}, "/bin/sh", false, false, nil, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_IgnoreErrorWithGlobalSilent(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true},
	}, "/bin/sh", true, false, nil, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_AllLinesIgnoreError(t *testing.T) {
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true},
		{Text: "exit 2", IgnoreError: true},
		{Text: "exit 127", IgnoreError: true},
	}, "/bin/sh", false, false, nil, io.Discard)
	assert.NoError(t, err)
}

func TestExecute_IgnoreErrorEchoStillShows(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "false", IgnoreError: true, Silent: false},
	}, "/bin/sh", false, false, nil, &buf)
	require.NoError(t, err)
	assert.Equal(t, "false\n", buf.String())
}

func TestExecute_IgnoreErrorCommandNotFound(t *testing.T) {
	// Shell returns exit code 127 for command not found — this is an ExitError,
	// so it should be ignored when IgnoreError is true.
	err := Execute([]ResolvedLine{
		{Text: "nonexistent-binary-xyz-12345", IgnoreError: true},
	}, "/bin/sh", false, false, nil, io.Discard)
	assert.NoError(t, err)
}

// --- DryRun behavior tests ---

func TestExecute_DryRunDoesNotExecute(t *testing.T) {
	// A failing command should not produce an error in dry-run mode.
	err := Execute(toLines("exit 1"), "/bin/sh", false, true, nil, io.Discard)
	assert.NoError(t, err)
}

func TestExecute_DryRunPrintsLines(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "echo hello"},
		{Text: "echo world"},
	}, "/bin/sh", false, true, nil, &buf)
	require.NoError(t, err)
	assert.Equal(t, "echo hello\necho world\n", buf.String())
}

func TestExecute_DryRunGlobalSilentSuppressesAll(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "echo hello"},
		{Text: "echo world"},
	}, "/bin/sh", true, true, nil, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestExecute_DryRunPerLineSilentSuppressed(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "echo visible"},
		{Text: "echo hidden", Silent: true},
	}, "/bin/sh", false, true, nil, &buf)
	require.NoError(t, err)
	assert.Equal(t, "echo visible\n", buf.String())
}

func TestExecute_DryRunEmptyLines(t *testing.T) {
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{}, "/bin/sh", false, true, nil, &buf)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// --- CommandArgs passthrough tests (Finding #7) ---

func TestExecute_CommandArgsPositional(t *testing.T) {
	// sh -c "echo $1 $2" -- hello world → file should contain "hello world"
	tmpFile := t.TempDir() + "/out.txt"
	err := Execute([]ResolvedLine{
		{Text: "echo $1 $2 > " + tmpFile},
	}, "/bin/sh", true, false, []string{"hello", "world"}, io.Discard)
	require.NoError(t, err)
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", string(content))
}

func TestExecute_CommandArgsAt(t *testing.T) {
	// Use a temp file to capture side-effects from the shell command.
	tmpFile := t.TempDir() + "/args.txt"
	err := Execute([]ResolvedLine{
		{Text: "echo $@ > " + tmpFile},
	}, "/bin/sh", true, false, []string{"a", "b", "c"}, io.Discard)
	require.NoError(t, err)
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "a b c\n", string(content))
}

func TestExecute_CommandArgsFirst(t *testing.T) {
	tmpFile := t.TempDir() + "/arg1.txt"
	err := Execute([]ResolvedLine{
		{Text: "echo $1 > " + tmpFile},
	}, "/bin/sh", true, false, []string{"first", "second"}, io.Discard)
	require.NoError(t, err)
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "first\n", string(content))
}

func TestExecute_EmptyCommandArgsNoEffect(t *testing.T) {
	// Passing nil/empty commandArgs should not affect commands that don't use $@.
	err := Execute(toLines("true"), "/bin/sh", false, false, nil, io.Discard)
	assert.NoError(t, err)

	err = Execute(toLines("true"), "/bin/sh", false, false, []string{}, io.Discard)
	assert.NoError(t, err)
}

func TestExecute_CommandArgsWithMultipleLines(t *testing.T) {
	tmpFile := t.TempDir() + "/out.txt"
	// Args should be available on every line in the sequence.
	err := Execute([]ResolvedLine{
		{Text: "echo $1 >> " + tmpFile},
		{Text: "echo $2 >> " + tmpFile},
	}, "/bin/sh", true, false, []string{"line1", "line2"}, io.Discard)
	require.NoError(t, err)
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2\n", string(content))
}

func TestExecute_CommandArgsDryRunDoesNotPassArgs(t *testing.T) {
	// In dry-run mode, no execution occurs so args don't matter —
	// only the echo output is produced.
	var buf bytes.Buffer
	err := Execute([]ResolvedLine{
		{Text: "echo $1"},
	}, "/bin/sh", false, true, []string{"unused"}, &buf)
	require.NoError(t, err)
	// dry-run echoes the unreplaced shell text, not the resolved arg value.
	assert.Equal(t, "echo $1\n", buf.String())
}

func TestExecute_CommandArgsWithSpacesInArg(t *testing.T) {
	// A single arg containing spaces should arrive in $1 as one unit.
	tmpFile := t.TempDir() + "/out.txt"
	err := Execute([]ResolvedLine{
		{Text: `echo "$1" > ` + tmpFile},
	}, "/bin/sh", true, false, []string{"hello world"}, io.Discard)
	require.NoError(t, err)
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", string(content))
}

func TestExecute_DoubleDashLinePrefixStrippedToSingleDash(t *testing.T) {
	// When an Opsfile line starts with "--something" the parser strips one "-"
	// leaving "-something" with IgnoreError=true (see TestParseOpsFile_DoubleDashPrefixParsed).
	// "-something" is not a valid shell command, so the shell returns exit 127.
	// With IgnoreError=true that failure is swallowed and execution continues.
	// This test documents the end-to-end behavior of that edge case.
	err := Execute([]ResolvedLine{
		{Text: "-this-is-not-a-command", IgnoreError: true},
		{Text: "true"},
	}, "/bin/sh", false, false, nil, io.Discard)
	assert.NoError(t, err, "IgnoreError should swallow the exit 127 from the unknown command")
}

func TestExecute_CommandArgsDollarZeroIsOps(t *testing.T) {
	// POSIX convention: sh -c "script" name arg1 arg2...
	// The string after the script becomes $0 inside the shell.
	// We use "ops" so users who reference $0 see a meaningful program name.
	tmpFile := t.TempDir() + "/out.txt"
	err := Execute([]ResolvedLine{
		{Text: "echo $0 > " + tmpFile},
	}, "/bin/sh", true, false, []string{"myapp"}, io.Discard)
	require.NoError(t, err)
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "ops\n", string(content))
}

// --- DefaultShell tests ---

func TestDefaultShell_ReturnsNonEmpty(t *testing.T) {
	shell := DefaultShell()
	assert.NotEmpty(t, shell)
}

func TestDefaultShell_FallbackWhenUnset(t *testing.T) {
	t.Setenv("SHELL", "")
	shell := DefaultShell()
	assert.Equal(t, "/bin/sh", shell)
}

func TestDefaultShell_UsesShellEnvVar(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	shell := DefaultShell()
	assert.Equal(t, "/bin/bash", shell)
}

func TestDefaultShell_ReturnsShellVerbatimEvenIfInvalid(t *testing.T) {
	// DefaultShell is a pure string reader — it returns whatever $SHELL
	// contains without validating the path. An invalid path will fail at
	// Execute time, not here.
	t.Setenv("SHELL", "/nonexistent/my-shell")
	shell := DefaultShell()
	assert.Equal(t, "/nonexistent/my-shell", shell)
}
