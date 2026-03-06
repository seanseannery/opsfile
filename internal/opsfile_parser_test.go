package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseExamples_AllFilesParseWithoutError(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	examplesDir := filepath.Join(filepath.Dir(thisFile), "..", "examples")

	files, err := filepath.Glob(filepath.Join(examplesDir, "Opsfile*"))
	if err != nil {
		t.Fatalf("globbing examples dir: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no example Opsfiles found")
	}

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			_, _, err := ParseOpsFile(path)
			if err != nil {
				t.Errorf("ParseOpsFile(%q) returned error: %v", filepath.Base(path), err)
			}
		})
	}
}


// writeTempOpsfile writes content to a temp file and returns its path.
func writeTempOpsfile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "Opsfile*")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

// examplesOpsfile returns the path to the examples/Opsfile relative to this test file.
func examplesOpsfile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	// internal/ -> repo root -> examples/Opsfile
	return filepath.Join(filepath.Dir(thisFile), "..", "examples", "Opsfile")
}

func TestParseOpsFile_Variables(t *testing.T) {
	vars, _, err := ParseOpsFile(examplesOpsfile(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct{ name, want string }{
		{"prod_AWS_ACCOUNT", "123456789012"},
		{"preprod_AWS_ACCOUNT", "987654321098"},
	}
	for _, tc := range cases {
		got, ok := vars[tc.name]
		if !ok {
			t.Errorf("variable %q not found", tc.name)
			continue
		}
		if got != tc.want {
			t.Errorf("variable %q: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestParseOpsFile_NoComments(t *testing.T) {
	vars, commands, err := ParseOpsFile(examplesOpsfile(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for k, v := range vars {
		if len(k) > 0 && k[0] == '#' {
			t.Errorf("comment leaked into variables: key=%q value=%q", k, v)
		}
	}
	for _, cmd := range commands {
		for env, lines := range cmd.Environments {
			for _, line := range lines {
				if len(line) > 0 && line[0] == '#' {
					t.Errorf("comment leaked into command %q env %q: %q", cmd.Name, env, line)
				}
			}
		}
	}
}

func TestParseOpsFile_Commands(t *testing.T) {
	_, commands, err := ParseOpsFile(examplesOpsfile(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCommands := []string{"tail-logs", "list-instance-ips", "show-profile"}
	for _, name := range expectedCommands {
		cmd, ok := commands[name]
		if !ok {
			t.Errorf("command %q not found", name)
			continue
		}
		if cmd.Name != name {
			t.Errorf("command Name field: got %q, want %q", cmd.Name, name)
		}
	}

	if got := len(commands); got != len(expectedCommands) {
		t.Errorf("expected %d commands, got %d", len(expectedCommands), got)
	}
}

func TestParseOpsFile_TailLogsEnvironments(t *testing.T) {
	_, commands, err := ParseOpsFile(examplesOpsfile(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd, ok := commands["tail-logs"]
	if !ok {
		t.Fatal("command 'tail-logs' not found")
	}

	// Should have "default" and "local" environments
	for _, env := range []string{"default", "local"} {
		if _, ok := cmd.Environments[env]; !ok {
			t.Errorf("environment %q not found in tail-logs", env)
		}
	}

	// default environment: 4 backslash-continuation lines join into 1
	defaultLines := cmd.Environments["default"]
	if len(defaultLines) != 1 {
		t.Errorf("tail-logs/default: expected 1 line, got %d: %v", len(defaultLines), defaultLines)
	} else if want := "aws logs tail $(LOG_GROUP) --follow --since 10m --region $(AWS_REGION)"; defaultLines[0] != want {
		t.Errorf("tail-logs/default line 0: got %q, want %q", defaultLines[0], want)
	}

	// local environment should have 1 shell line
	localLines := cmd.Environments["local"]
	if len(localLines) != 1 {
		t.Errorf("tail-logs/local: expected 1 line, got %d: %v", len(localLines), localLines)
	} else if want := "docker logs my-service --follow --tail 100"; localLines[0] != want {
		t.Errorf("tail-logs/local line 0: got %q, want %q", localLines[0], want)
	}
}

func TestParseOpsFile_ListInstanceIpsEnvironments(t *testing.T) {
	_, commands, err := ParseOpsFile(examplesOpsfile(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd, ok := commands["list-instance-ips"]
	if !ok {
		t.Fatal("command 'list-instance-ips' not found")
	}

	for _, env := range []string{"prod", "preprod"} {
		if _, ok := cmd.Environments[env]; !ok {
			t.Errorf("environment %q not found in list-instance-ips", env)
		}
	}

	// Both environments use backslash continuation — each produces 1 joined shell line.
	prodLines := cmd.Environments["prod"]
	if len(prodLines) != 1 {
		t.Errorf("list-instance-ips/prod: expected 1 line, got %d: %v", len(prodLines), prodLines)
	}

	preprodLines := cmd.Environments["preprod"]
	if len(preprodLines) != 1 {
		t.Errorf("list-instance-ips/preprod: expected 1 line, got %d: %v", len(preprodLines), preprodLines)
	}
}

func TestParseOpsFile_FileNotFound(t *testing.T) {
	_, _, err := ParseOpsFile("/nonexistent/path/Opsfile")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestExtractVariableValue(t *testing.T) {
	okCases := []struct {
		raw  string
		want string
	}{
		// Unquoted: inline comment stripped (single space, multiple spaces, tab)
		{"123 # a comment", "123"},
		{"hello world # note", "hello world"},
		{"123   # multiple spaces", "123"},
		{"123\t# tab before hash", "123"},
		// Unquoted: # without preceding whitespace is part of the value
		{"val#nospace", "val#nospace"},
		// Unquoted: no comment
		{"plainvalue", "plainvalue"},
		// Double-quoted: # inside preserved, closing quote ends value
		{`"my#value"`, "my#value"},
		{`"my#value" # comment`, "my#value"},
		// Single-quoted: same rules
		{"'single#quotes'", "single#quotes"},
		{"'single#quotes' # comment", "single#quotes"},
		// Empty quoted string
		{`""`, ""},
	}

	for _, tc := range okCases {
		got, err := extractVariableValue(tc.raw)
		if err != nil {
			t.Errorf("extractVariableValue(%q) unexpected error: %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("extractVariableValue(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}

	errorCases := []string{
		`"no closing double quote`,
		`'no closing single quote`,
	}
	for _, raw := range errorCases {
		_, err := extractVariableValue(raw)
		if err == nil {
			t.Errorf("extractVariableValue(%q) expected error, got nil", raw)
		}
	}
}

func TestParseOpsFile_InlineCommentOnVariable(t *testing.T) {
	content := `
PLAIN=123 # inline comment
TABS=456   # multiple spaces
QUOTED="has#hash" # another comment
NOSPACE=val#notcomment

my-cmd:
    prod:
        aws something
`
	vars, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cases := []struct{ name, want string }{
		{"PLAIN", "123"},
		{"TABS", "456"},
		{"QUOTED", "has#hash"},
		{"NOSPACE", "val#notcomment"},
	}
	for _, tc := range cases {
		if got := vars[tc.name]; got != tc.want {
			t.Errorf("variable %q: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestParseOpsFile_UnclosedQuote(t *testing.T) {
	content := `
MY_VAR="unclosed

my-cmd:
    prod:
        aws something
`
	_, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err == nil {
		t.Fatal("expected error for unclosed quote, got nil")
	}
	if !strings.Contains(err.Error(), "unclosed") {
		t.Errorf("error should mention 'unclosed', got: %v", err)
	}
}

func TestParseOpsFile_EmptyFile(t *testing.T) {
	_, _, err := ParseOpsFile(writeTempOpsfile(t, ""))
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got: %v", err)
	}
}

func TestParseOpsFile_OnlyComments(t *testing.T) {
	content := "# just a comment\n# another comment\n"
	_, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err == nil {
		t.Fatal("expected error for comment-only file, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got: %v", err)
	}
}

func TestParseOpsFile_DuplicateCommand(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws ec2 something

my-cmd:
    preprod:
        aws ecs something
`
	_, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err == nil {
		t.Fatal("expected error for duplicate command, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention 'duplicate', got: %v", err)
	}
}

func TestParseOpsFile_VariableMissingName(t *testing.T) {
	content := `
=somevalue

my-cmd:
    prod:
        aws something
`
	_, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err == nil {
		t.Fatal("expected error for variable with missing name, got nil")
	}
	if !strings.Contains(err.Error(), "missing name") {
		t.Errorf("error should mention 'missing name', got: %v", err)
	}
}

func TestParseOpsFile_BackslashContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs \
            --log-group /my/group
`
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := commands["my-cmd"].Environments["prod"]
	if len(lines) != 1 {
		t.Fatalf("expected 1 shell line, got %d: %v", len(lines), lines)
	}
	want := "aws cloudwatch logs --log-group /my/group"
	if lines[0] != want {
		t.Errorf("got %q, want %q", lines[0], want)
	}
}

func TestParseOpsFile_BackslashContinuationChain(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs \
            --log-group /my/group \
            --tail
`
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := commands["my-cmd"].Environments["prod"]
	if len(lines) != 1 {
		t.Fatalf("expected 1 shell line, got %d: %v", len(lines), lines)
	}
	want := "aws cloudwatch logs --log-group /my/group --tail"
	if lines[0] != want {
		t.Errorf("got %q, want %q", lines[0], want)
	}
}

func TestParseOpsFile_BackslashSpaceBeforeSlash(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws logs \
            --tail
`
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := commands["my-cmd"].Environments["prod"]
	if len(lines) != 1 {
		t.Fatalf("expected 1 shell line, got %d: %v", len(lines), lines)
	}
	want := "aws logs --tail"
	if lines[0] != want {
		t.Errorf("got %q, want %q", lines[0], want)
	}
}

func TestParseOpsFile_IndentContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs
            --log-group /my/group
`
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := commands["my-cmd"].Environments["prod"]
	if len(lines) != 1 {
		t.Fatalf("expected 1 shell line, got %d: %v", len(lines), lines)
	}
	want := "aws cloudwatch logs --log-group /my/group"
	if lines[0] != want {
		t.Errorf("got %q, want %q", lines[0], want)
	}
}

func TestParseOpsFile_IndentContinuationChain(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs
            --log-group /my/group
            --tail
`
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := commands["my-cmd"].Environments["prod"]
	if len(lines) != 1 {
		t.Fatalf("expected 1 shell line, got %d: %v", len(lines), lines)
	}
	want := "aws cloudwatch logs --log-group /my/group --tail"
	if lines[0] != want {
		t.Errorf("got %q, want %q", lines[0], want)
	}
}

func TestParseOpsFile_IndentNewCommandAfterContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs
            --log-group /my/group
        echo done
`
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := commands["my-cmd"].Environments["prod"]
	if len(lines) != 2 {
		t.Fatalf("expected 2 shell lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "aws cloudwatch logs --log-group /my/group" {
		t.Errorf("line 0: got %q", lines[0])
	}
	if lines[1] != "echo done" {
		t.Errorf("line 1: got %q", lines[1])
	}
}

func TestParseOpsFile_VariableWhitespaceOnlyValue(t *testing.T) {
	content := "VAR=   \n\nmy-cmd:\n    prod:\n        echo hello\n"
	vars, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// TrimSpace on the value after "=" yields empty string.
	if got := vars["VAR"]; got != "" {
		t.Errorf("VAR: got %q, want %q", got, "")
	}
}

func TestParseOpsFile_VariableEmptyUnquotedValue(t *testing.T) {
	content := `
VAR=

my-cmd:
    prod:
        echo hello
`
	vars, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := vars["VAR"]; got != "" {
		t.Errorf("VAR: got %q, want %q", got, "")
	}
}

func TestParseOpsFile_MultipleEnvironments(t *testing.T) {
	content := `
my-cmd:
    prod:
        echo prod
    preprod:
        echo preprod
    local:
        echo local
    default:
        echo default
`
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := commands["my-cmd"]
	for _, env := range []string{"prod", "preprod", "local", "default"} {
		lines, ok := cmd.Environments[env]
		if !ok {
			t.Errorf("environment %q not found", env)
			continue
		}
		want := "echo " + env
		if len(lines) != 1 || lines[0] != want {
			t.Errorf("env %q: got %v, want [%q]", env, lines, want)
		}
	}
}

func TestParseOpsFile_MixedBackslashAndIndentContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws logs tail \
            --follow
            --since 10m
`
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := commands["my-cmd"].Environments["prod"]
	// Backslash joins first two into one line; the third line at same indent
	// is NOT deeper than the flushed continuation, so it's a separate shell line.
	if len(lines) != 2 {
		t.Fatalf("expected 2 shell lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "aws logs tail --follow" {
		t.Errorf("line 0: got %q, want %q", lines[0], "aws logs tail --follow")
	}
	if lines[1] != "--since 10m" {
		t.Errorf("line 1: got %q, want %q", lines[1], "--since 10m")
	}
}

func TestParseOpsFile_TabIndentedShellLines(t *testing.T) {
	content := "my-cmd:\n\tprod:\n\t\techo hello\n\t\techo world\n"
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := commands["my-cmd"].Environments["prod"]
	if len(lines) != 2 {
		t.Fatalf("expected 2 shell lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "echo hello" {
		t.Errorf("line 0: got %q, want %q", lines[0], "echo hello")
	}
	if lines[1] != "echo world" {
		t.Errorf("line 1: got %q, want %q", lines[1], "echo world")
	}
}

func TestParseOpsFile_CommandHeaderTrailingWhitespace(t *testing.T) {
	// Trailing spaces after ":" should still parse as a command header.
	// Note: TrimSpace strips trailing whitespace, so "my-cmd:  " -> "my-cmd:"
	content := "my-cmd:  \n    prod:\n        echo hello\n"
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := commands["my-cmd"]; !ok {
		t.Error("command 'my-cmd' not found")
	}
}

func TestParseOpsFile_EnvHeaderTrailingWhitespace(t *testing.T) {
	// Trailing whitespace on env header line.
	content := "my-cmd:\n    prod:  \n        echo hello\n"
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := commands["my-cmd"].Environments["prod"]; !ok {
		t.Error("environment 'prod' not found")
	}
}

func TestParseOpsFile_VariableValueLeadingWhitespace(t *testing.T) {
	content := `
VAR=  value

my-cmd:
    prod:
        echo hello
`
	vars, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// parseVariable does TrimSpace on the value after "=", so "  value" -> "value"
	if got := vars["VAR"]; got != "value" {
		t.Errorf("VAR: got %q, want %q", got, "value")
	}
}

func TestParseOpsFile_LineNumberInParseError(t *testing.T) {
	content := `
GOOD=fine
BAD="unclosed

my-cmd:
    prod:
        echo hello
`
	_, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err == nil {
		t.Fatal("expected error for unclosed quote, got nil")
	}
	// The unclosed quote is on line 3 (blank line 1, GOOD=fine line 2, BAD=... line 3).
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("error should include line number, got: %v", err)
	}
}

func TestParseOpsFile_BackslashTrailingEOF(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs \`
	_, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := commands["my-cmd"].Environments["prod"]
	if len(lines) != 1 {
		t.Fatalf("expected 1 shell line, got %d: %v", len(lines), lines)
	}
	// The trailing \ is stripped; what remains is the fragment before it.
	want := "aws cloudwatch logs "
	if lines[0] != want {
		t.Errorf("got %q, want %q", lines[0], want)
	}
}
