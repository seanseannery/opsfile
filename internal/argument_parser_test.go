package internal

import (
	"strings"
	"testing"
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
			wantErrSub: "flag provided but not defined",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotFlags, gotPos, err := ParseOpsFlags(tc.input)

			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("error: got %v, want %v", err, tc.wantErr)
				}
				return
			}
			if tc.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrSub)
				}
				if !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotFlags != tc.wantFlags {
				t.Errorf("OpsFlags: got %+v, want %+v", gotFlags, tc.wantFlags)
			}
			if len(gotPos) != len(tc.wantPos) {
				t.Fatalf("positionals: got %v, want %v", gotPos, tc.wantPos)
			}
			for i, w := range tc.wantPos {
				if gotPos[i] != w {
					t.Errorf("positionals[%d]: got %q, want %q", i, gotPos[i], w)
				}
			}
		})
	}
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
			name:     "env, command, and passthrough args",
			input:    []string{"prod", "tail-logs", "--since", "1h"},
			wantEnv:  "prod", wantCmd: "tail-logs",
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseOpsArgs(tc.input)
			if tc.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrSub)
				}
				if !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.OpsEnv != tc.wantEnv {
				t.Errorf("OpsEnv: got %q, want %q", got.OpsEnv, tc.wantEnv)
			}
			if got.OpsCommand != tc.wantCmd {
				t.Errorf("OpsCommand: got %q, want %q", got.OpsCommand, tc.wantCmd)
			}
			if len(tc.wantArgs) == 0 && len(got.CommandArgs) == 0 {
				return
			}
			if len(got.CommandArgs) != len(tc.wantArgs) {
				t.Errorf("CommandArgs: got %v, want %v", got.CommandArgs, tc.wantArgs)
				return
			}
			for i, a := range tc.wantArgs {
				if got.CommandArgs[i] != a {
					t.Errorf("CommandArgs[%d]: got %q, want %q", i, got.CommandArgs[i], a)
				}
			}
		})
	}
}
