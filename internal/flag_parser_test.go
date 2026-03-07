package internal

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOpsFlags(t *testing.T) {
	cases := []struct {
		name       string
		input      []string
		wantFlags  OpsFlags
		wantPos    []string
		wantErr    error  // exact sentinel, or nil
		wantErrSub string // substring match for non-sentinel errors
	}{
		{
			name:      "no flags, positionals passed through",
			input:     []string{"prod", "tail-logs"},
			wantFlags: OpsFlags{},
			wantPos:   []string{"prod", "tail-logs"},
		},
		{
			name:      "empty input",
			input:     []string{},
			wantFlags: OpsFlags{},
			wantPos:   []string{},
		},
		{
			name:      "-D sets Directory",
			input:     []string{"-D", "/some/path", "prod", "cmd"},
			wantFlags: OpsFlags{Directory: "/some/path"},
			wantPos:   []string{"prod", "cmd"},
		},
		{
			name:      "--directory sets Directory",
			input:     []string{"--directory", "/some/path", "prod", "cmd"},
			wantFlags: OpsFlags{Directory: "/some/path"},
			wantPos:   []string{"prod", "cmd"},
		},
		{
			name:      "-d sets DryRun",
			input:     []string{"-d", "prod", "cmd"},
			wantFlags: OpsFlags{DryRun: true},
			wantPos:   []string{"prod", "cmd"},
		},
		{
			name:      "--dry-run sets DryRun",
			input:     []string{"--dry-run", "prod", "cmd"},
			wantFlags: OpsFlags{DryRun: true},
			wantPos:   []string{"prod", "cmd"},
		},
		{
			name:      "-s sets Silent",
			input:     []string{"-s", "prod", "cmd"},
			wantFlags: OpsFlags{Silent: true},
			wantPos:   []string{"prod", "cmd"},
		},
		{
			name:      "--silent sets Silent",
			input:     []string{"--silent", "prod", "cmd"},
			wantFlags: OpsFlags{Silent: true},
			wantPos:   []string{"prod", "cmd"},
		},
		{
			name:      "-v sets Version",
			input:     []string{"-v"},
			wantFlags: OpsFlags{Version: true},
			wantPos:   []string{},
		},
		{
			name:      "--version sets Version",
			input:     []string{"--version"},
			wantFlags: OpsFlags{Version: true},
			wantPos:   []string{},
		},
		{
			name:    "-h returns ErrHelp",
			input:   []string{"-h"},
			wantErr: ErrHelp,
		},
		{
			name:    "--help returns ErrHelp",
			input:   []string{"--help"},
			wantErr: ErrHelp,
		},
		{
			name:    "-? returns ErrHelp",
			input:   []string{"-?"},
			wantErr: ErrHelp,
		},
		{
			name:       "unknown flag returns error",
			input:      []string{"-z"},
			wantErrSub: "unknown shorthand flag",
		},
		{
			name:      "multiple flags combined -d -s",
			input:     []string{"-d", "-s", "prod", "cmd"},
			wantFlags: OpsFlags{DryRun: true, Silent: true},
			wantPos:   []string{"prod", "cmd"},
		},
		{
			name:       "-D with missing argument",
			input:      []string{"-D"},
			wantErrSub: "flag needs an argument",
		},
		{
			name:       "unknown long flag --foobar",
			input:      []string{"--foobar"},
			wantErrSub: "unknown flag: --foobar",
		},
		{
			name:      "flag after positional treated as positional",
			input:     []string{"prod", "-d", "cmd"},
			wantFlags: OpsFlags{},
			wantPos:   []string{"prod", "-d", "cmd"},
		},
		{
			name:    "-? combined with other args still returns ErrHelp",
			input:   []string{"-?", "prod", "cmd"},
			wantErr: ErrHelp,
		},
		{
			name:      "-D with empty string value",
			input:     []string{"-D", "", "prod", "cmd"},
			wantFlags: OpsFlags{Directory: ""},
			wantPos:   []string{"prod", "cmd"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotFlags, gotPos, err := ParseOpsFlags(tc.input, io.Discard)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			if tc.wantErrSub != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErrSub)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantFlags, gotFlags)
			assert.Equal(t, tc.wantPos, gotPos)
		})
	}
}

func TestParseOpsFlags_HelpOutput(t *testing.T) {
	var buf bytes.Buffer
	_, _, gotErr := ParseOpsFlags([]string{"-h"}, &buf)

	require.ErrorIs(t, gotErr, ErrHelp)

	output := buf.String()
	for _, want := range []string{"-D", "-d", "-s", "-v"} {
		assert.Contains(t, output, want)
	}

	// Verify unknown flag error includes the flag name.
	_, _, unknownErr := ParseOpsFlags([]string{"--foobar"}, &buf)
	require.Error(t, unknownErr)
	assert.ErrorContains(t, unknownErr, "foobar")
}

func TestParseOpsArgs(t *testing.T) {
	cases := []struct {
		name       string
		input      []string
		wantEnv    string
		wantCmd    string
		wantArgs   []string
		wantErrSub string
	}{
		{
			name:    "env and command",
			input:   []string{"prod", "tail-logs"},
			wantEnv: "prod", wantCmd: "tail-logs",
		},
		{
			name:    "env, command, and passthrough args",
			input:   []string{"prod", "tail-logs", "--since", "1h"},
			wantEnv: "prod", wantCmd: "tail-logs",
			wantArgs: []string{"--since", "1h"},
		},
		{
			name:       "no args",
			input:      []string{},
			wantErrSub: "missing environment",
		},
		{
			name:       "only env",
			input:      []string{"prod"},
			wantErrSub: "missing command",
		},
		{
			name:     "many passthrough args preserved in order",
			input:    []string{"prod", "cmd", "arg1", "arg2", "arg3"},
			wantEnv:  "prod",
			wantCmd:  "cmd",
			wantArgs: []string{"arg1", "arg2", "arg3"},
		},
		{
			name:    "hyphenated env and command names",
			input:   []string{"my-env", "my-cmd"},
			wantEnv: "my-env",
			wantCmd: "my-cmd",
		},
		{
			name:    "empty string in positionals",
			input:   []string{"", "cmd"},
			wantEnv: "",
			wantCmd: "cmd",
		},
		{
			name:     "flag-like values in passthrough are not consumed",
			input:    []string{"prod", "cmd", "--verbose", "-n", "5"},
			wantEnv:  "prod",
			wantCmd:  "cmd",
			wantArgs: []string{"--verbose", "-n", "5"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseOpsArgs(tc.input)
			if tc.wantErrSub != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErrSub)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantEnv, got.OpsEnv)
			assert.Equal(t, tc.wantCmd, got.OpsCommand)
			if len(tc.wantArgs) == 0 {
				assert.Empty(t, got.CommandArgs)
			} else {
				assert.Equal(t, tc.wantArgs, got.CommandArgs)
			}
		})
	}
}
