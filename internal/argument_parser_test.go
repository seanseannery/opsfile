package internal

import (
	"strings"
	"testing"
)

func TestParseOpsFlags(t *testing.T) {
	cases := []struct {
		name        string
		input       []string
		wantPos     []string
		wantErrSub  string
	}{
		{
			name:    "no flags, positionals passed through",
			input:   []string{"prod", "tail-logs"},
			wantPos: []string{"prod", "tail-logs"},
		},
		{
			name:    "empty input",
			input:   []string{},
			wantPos: []string{},
		},
		{
			name:       "unknown flag",
			input:      []string{"-unknown-flag"},
			wantErrSub: "flag provided but not defined",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseOpsFlags(tc.input)
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
			if len(got) != len(tc.wantPos) {
				t.Fatalf("positionals: got %v, want %v", got, tc.wantPos)
			}
			for i, w := range tc.wantPos {
				if got[i] != w {
					t.Errorf("positionals[%d]: got %q, want %q", i, got[i], w)
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
