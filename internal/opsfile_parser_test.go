package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExamples_AllFilesParseWithoutError(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	examplesDir := filepath.Join(filepath.Dir(thisFile), "..", "examples")

	files, err := filepath.Glob(filepath.Join(examplesDir, "Opsfile*"))
	require.NoError(t, err, "globbing examples dir")
	require.NotEmpty(t, files, "no example Opsfiles found")

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			_, err := ParseOpsFile(path)
			assert.NoError(t, err)
		})
	}
}

// writeTempOpsfile writes content to a temp file and returns its path.
func writeTempOpsfile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "Opsfile*")
	require.NoError(t, err, "creating temp file")
	_, err = f.WriteString(content)
	require.NoError(t, err, "writing temp file")
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
	parsed, err := ParseOpsFile(examplesOpsfile(t))
	require.NoError(t, err)

	cases := []struct{ name, want string }{
		{"prod_AWS_ACCOUNT", "123456789012"},
		{"preprod_AWS_ACCOUNT", "987654321098"},
	}
	for _, tc := range cases {
		got, ok := parsed.Variables[tc.name]
		require.True(t, ok, "variable %q not found", tc.name)
		assert.Equal(t, tc.want, got, "variable %q", tc.name)
	}
}

func TestParseOpsFile_NoComments(t *testing.T) {
	parsed, err := ParseOpsFile(examplesOpsfile(t))
	require.NoError(t, err)

	for k, v := range parsed.Variables {
		if len(k) > 0 && k[0] == '#' {
			t.Errorf("comment leaked into variables: key=%q value=%q", k, v)
		}
	}
	for _, cmd := range parsed.Commands {
		for env, lines := range cmd.Environments {
			for _, line := range lines {
				if len(line.Text) > 0 && line.Text[0] == '#' {
					t.Errorf("comment leaked into command %q env %q: %q", cmd.Name, env, line.Text)
				}
			}
		}
	}
}

func TestParseOpsFile_Commands(t *testing.T) {
	parsed, err := ParseOpsFile(examplesOpsfile(t))
	require.NoError(t, err)

	expectedCommands := []string{"tail-logs", "list-instance-ips", "show-profile", "redeploy"}
	for _, name := range expectedCommands {
		cmd, ok := parsed.Commands[name]
		require.True(t, ok, "command %q not found", name)
		assert.Equal(t, name, cmd.Name, "command Name field")
	}

	assert.Len(t, parsed.Commands, len(expectedCommands))
}

func TestParseOpsFile_TailLogsEnvironments(t *testing.T) {
	parsed, err := ParseOpsFile(examplesOpsfile(t))
	require.NoError(t, err)

	cmd, ok := parsed.Commands["tail-logs"]
	require.True(t, ok, "command 'tail-logs' not found")

	// Should have "default" and "local" environments
	for _, env := range []string{"default", "local"} {
		assert.Contains(t, cmd.Environments, env, "environment %q not found in tail-logs", env)
	}

	// default environment: 4 backslash-continuation lines join into 1
	defaultLines := cmd.Environments["default"]
	require.Len(t, defaultLines, 1, "tail-logs/default: expected 1 line")
	assert.Equal(t, "aws logs tail $(LOG_GROUP) --follow --since 10m --region $(AWS_REGION)", defaultLines[0].Text)

	// local environment should have 1 shell line
	localLines := cmd.Environments["local"]
	require.Len(t, localLines, 1, "tail-logs/local: expected 1 line")
	assert.Equal(t, "docker logs my-service --follow --tail 100", localLines[0].Text)
}

func TestParseOpsFile_ListInstanceIpsEnvironments(t *testing.T) {
	parsed, err := ParseOpsFile(examplesOpsfile(t))
	require.NoError(t, err)

	cmd, ok := parsed.Commands["list-instance-ips"]
	require.True(t, ok, "command 'list-instance-ips' not found")

	for _, env := range []string{"prod", "preprod"} {
		assert.Contains(t, cmd.Environments, env, "environment %q not found in list-instance-ips", env)
	}

	// Both environments use backslash continuation — each produces 1 joined shell line.
	prodLines := cmd.Environments["prod"]
	assert.Len(t, prodLines, 1, "list-instance-ips/prod: expected 1 line")

	preprodLines := cmd.Environments["preprod"]
	assert.Len(t, preprodLines, 1, "list-instance-ips/preprod: expected 1 line")
}

func TestParseOpsFile_FileNotFound(t *testing.T) {
	_, err := ParseOpsFile("/nonexistent/path/Opsfile")
	require.Error(t, err)
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
		require.NoError(t, err, "extractVariableValue(%q) unexpected error", tc.raw)
		assert.Equal(t, tc.want, got, "extractVariableValue(%q)", tc.raw)
	}

	errorCases := []string{
		`"no closing double quote`,
		`'no closing single quote`,
	}
	for _, raw := range errorCases {
		_, err := extractVariableValue(raw)
		assert.Error(t, err, "extractVariableValue(%q) expected error", raw)
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
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	cases := []struct{ name, want string }{
		{"PLAIN", "123"},
		{"TABS", "456"},
		{"QUOTED", "has#hash"},
		{"NOSPACE", "val#notcomment"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, parsed.Variables[tc.name], "variable %q", tc.name)
	}
}

func TestParseOpsFile_UnclosedQuote(t *testing.T) {
	content := `
MY_VAR="unclosed

my-cmd:
    prod:
        aws something
`
	_, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unclosed")
}

func TestParseOpsFile_EmptyFile(t *testing.T) {
	_, err := ParseOpsFile(writeTempOpsfile(t, ""))
	require.Error(t, err)
	assert.ErrorContains(t, err, "empty")
}

func TestParseOpsFile_OnlyComments(t *testing.T) {
	content := "# just a comment\n# another comment\n"
	_, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.Error(t, err)
	assert.ErrorContains(t, err, "empty")
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
	_, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.Error(t, err)
	assert.ErrorContains(t, err, "duplicate")
}

func TestParseOpsFile_VariableMissingName(t *testing.T) {
	content := `
=somevalue

my-cmd:
    prod:
        aws something
`
	_, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing name")
}

func TestParseOpsFile_BackslashContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs \
            --log-group /my/group
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "aws cloudwatch logs --log-group /my/group", lines[0].Text)
}

func TestParseOpsFile_BackslashContinuationChain(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs \
            --log-group /my/group \
            --tail
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "aws cloudwatch logs --log-group /my/group --tail", lines[0].Text)
}

func TestParseOpsFile_BackslashSpaceBeforeSlash(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws logs \
            --tail
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "aws logs --tail", lines[0].Text)
}

func TestParseOpsFile_IndentContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs
            --log-group /my/group
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "aws cloudwatch logs --log-group /my/group", lines[0].Text)
}

func TestParseOpsFile_IndentContinuationChain(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs
            --log-group /my/group
            --tail
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "aws cloudwatch logs --log-group /my/group --tail", lines[0].Text)
}

func TestParseOpsFile_IndentNewCommandAfterContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs
            --log-group /my/group
        echo done
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 2)
	assert.Equal(t, "aws cloudwatch logs --log-group /my/group", lines[0].Text)
	assert.Equal(t, "echo done", lines[1].Text)
}

func TestParseOpsFile_VariableWhitespaceOnlyValue(t *testing.T) {
	content := "VAR=   \n\nmy-cmd:\n    prod:\n        echo hello\n"
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	// TrimSpace on the value after "=" yields empty string.
	assert.Equal(t, "", parsed.Variables["VAR"])
}

func TestParseOpsFile_VariableEmptyUnquotedValue(t *testing.T) {
	content := `
VAR=

my-cmd:
    prod:
        echo hello
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	assert.Equal(t, "", parsed.Variables["VAR"])
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
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	cmd := parsed.Commands["my-cmd"]
	for _, env := range []string{"prod", "preprod", "local", "default"} {
		lines, ok := cmd.Environments[env]
		require.True(t, ok, "environment %q not found", env)
		assert.Equal(t, []ShellLine{{Text: "echo " + env}}, lines, "env %q", env)
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
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	// Backslash joins first two into one line; the third line at same indent
	// is NOT deeper than the flushed continuation, so it's a separate shell line.
	require.Len(t, lines, 2)
	assert.Equal(t, "aws logs tail --follow", lines[0].Text)
	// "--since 10m" starts with '-' so the parser strips it as IgnoreError prefix,
	// leaving "-since 10m" as the shell text — same end result as before (resolver
	// previously did this stripping).
	assert.Equal(t, "-since 10m", lines[1].Text)
	assert.True(t, lines[1].IgnoreError)
}

func TestParseOpsFile_TabIndentedShellLines(t *testing.T) {
	content := "my-cmd:\n\tprod:\n\t\techo hello\n\t\techo world\n"
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 2)
	assert.Equal(t, "echo hello", lines[0].Text)
	assert.Equal(t, "echo world", lines[1].Text)
}

func TestParseOpsFile_CommandHeaderTrailingWhitespace(t *testing.T) {
	// Trailing spaces after ":" should still parse as a command header.
	// Note: TrimSpace strips trailing whitespace, so "my-cmd:  " -> "my-cmd:"
	content := "my-cmd:  \n    prod:\n        echo hello\n"
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	assert.Contains(t, parsed.Commands, "my-cmd")
}

func TestParseOpsFile_EnvHeaderTrailingWhitespace(t *testing.T) {
	// Trailing whitespace on env header line.
	content := "my-cmd:\n    prod:  \n        echo hello\n"
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	assert.Contains(t, parsed.Commands["my-cmd"].Environments, "prod")
}

func TestParseOpsFile_VariableValueLeadingWhitespace(t *testing.T) {
	content := `
VAR=  value

my-cmd:
    prod:
        echo hello
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	// parseVariable does TrimSpace on the value after "=", so "  value" -> "value"
	assert.Equal(t, "value", parsed.Variables["VAR"])
}

func TestParseOpsFile_LineNumberInParseError(t *testing.T) {
	content := `
GOOD=fine
BAD="unclosed

my-cmd:
    prod:
        echo hello
`
	_, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.Error(t, err)
	// The unclosed quote is on line 3 (blank line 1, GOOD=fine line 2, BAD=... line 3).
	assert.ErrorContains(t, err, "line 3")
}

func TestParseOpsFile_BackslashTrailingEOF(t *testing.T) {
	content := `
my-cmd:
    prod:
        aws cloudwatch logs \`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	// The trailing \ is stripped; what remains is the fragment before it.
	assert.Equal(t, "aws cloudwatch logs ", lines[0].Text)
}

func TestParseOpsFile_Description(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		wantDesc string
	}{
		{
			name: "comment directly above command",
			content: `# Deploy the service
deploy:
    prod:
        echo deploying
`,
			wantDesc: "Deploy the service",
		},
		{
			name: "blank line between comment and command",
			content: `# Deploy the service

deploy:
    prod:
        echo deploying
`,
			wantDesc: "",
		},
		{
			name: "multi-line comments capture first line of block",
			content: `# First line of comments
# Second line of comments
# Last line before command
deploy:
    prod:
        echo deploying
`,
			wantDesc: "First line of comments",
		},
		{
			name: "no comment yields empty description",
			content: `deploy:
    prod:
        echo deploying
`,
			wantDesc: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseOpsFile(writeTempOpsfile(t, tc.content))
			require.NoError(t, err)
			assert.Equal(t, tc.wantDesc, parsed.Commands["deploy"].Description)
		})
	}
}

func TestParseOpsFile_CommandOrder(t *testing.T) {
	content := `
# First command
alpha:
    default:
        echo alpha

# Second command
charlie:
    default:
        echo charlie

bravo:
    default:
        echo bravo
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "charlie", "bravo"}, parsed.CommandOrder)
}

func TestParseOpsFile_EnvOrder(t *testing.T) {
	content := `
cmd-a:
    prod:
        echo a-prod
    local:
        echo a-local

cmd-b:
    local:
        echo b-local
    preprod:
        echo b-preprod
    prod:
        echo b-prod
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	// Deduplicated, first-appearance order: prod, local, preprod
	assert.Equal(t, []string{"prod", "local", "preprod"}, parsed.EnvOrder)
}

// --- ShellLine prefix parsing tests ---

func TestParseOpsFile_AtPrefixParsed(t *testing.T) {
	content := `
my-cmd:
    prod:
        @echo hello
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "echo hello", lines[0].Text)
	assert.True(t, lines[0].Silent)
	assert.False(t, lines[0].IgnoreError)
}

func TestParseOpsFile_DashPrefixParsed(t *testing.T) {
	content := `
my-cmd:
    prod:
        -echo hello
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "echo hello", lines[0].Text)
	assert.False(t, lines[0].Silent)
	assert.True(t, lines[0].IgnoreError)
}

func TestParseOpsFile_AtDashPrefixParsed(t *testing.T) {
	content := `
my-cmd:
    prod:
        @-echo hello
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "echo hello", lines[0].Text)
	assert.True(t, lines[0].Silent)
	assert.True(t, lines[0].IgnoreError)
}

func TestParseOpsFile_DashAtPrefixParsed(t *testing.T) {
	content := `
my-cmd:
    prod:
        -@echo hello
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "echo hello", lines[0].Text)
	assert.True(t, lines[0].Silent)
	assert.True(t, lines[0].IgnoreError)
}

func TestParseOpsFile_NoPrefixParsed(t *testing.T) {
	content := `
my-cmd:
    prod:
        echo hello
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "echo hello", lines[0].Text)
	assert.False(t, lines[0].Silent)
	assert.False(t, lines[0].IgnoreError)
}

func TestParseOpsFile_AtPrefixWithBackslashContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        @aws logs \
            --follow
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "aws logs --follow", lines[0].Text)
	assert.True(t, lines[0].Silent)
}

func TestParseOpsFile_AtPrefixWithIndentContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        @aws logs
            --follow
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "aws logs --follow", lines[0].Text)
	assert.True(t, lines[0].Silent)
}

func TestParseOpsFile_AtOnContinuationFragment(t *testing.T) {
	// @ on a non-first continuation fragment is part of the joined shell text,
	// not Opsfile syntax (per FR-5).
	content := `
my-cmd:
    prod:
        aws logs \
            @--follow
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "aws logs @--follow", lines[0].Text)
	assert.False(t, lines[0].Silent)
}

func TestParseOpsFile_DashPrefixWithBackslashContinuation(t *testing.T) {
	content := `
my-cmd:
    prod:
        -aws logs \
            --follow
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	assert.Equal(t, "aws logs --follow", lines[0].Text)
	assert.True(t, lines[0].IgnoreError)
}

func TestParseOpsFile_DoubleAtPrefixParsed(t *testing.T) {
	content := `
my-cmd:
    prod:
        @@echo hello
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	// First @ sets silent, second @ is part of the text.
	assert.Equal(t, "@echo hello", lines[0].Text)
	assert.True(t, lines[0].Silent)
}

func TestParseOpsFile_DoubleDashPrefixParsed(t *testing.T) {
	content := `
my-cmd:
    prod:
        --echo hello
`
	parsed, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	lines := parsed.Commands["my-cmd"].Environments["prod"]
	require.Len(t, lines, 1)
	// First - sets ignoreError, second - is part of the text.
	assert.Equal(t, "-echo hello", lines[0].Text)
	assert.True(t, lines[0].IgnoreError)
}

// --- FindOpsfileDir tests ---

// realPath resolves symlinks so tests work on macOS where /var -> /private/var.
func realPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err, "EvalSymlinks(%q)", path)
	return resolved
}

func TestFindOpsfileDir_FoundInCwd(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	err := os.WriteFile(filepath.Join(tmp, OpsFileName), []byte(""), 0o644)
	require.NoError(t, err)

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(tmp)

	got, err := FindOpsfileDir()
	require.NoError(t, err)
	assert.Equal(t, tmp, got)
}

func TestFindOpsfileDir_FoundInParent(t *testing.T) {
	parent := realPath(t, t.TempDir())
	child := filepath.Join(parent, "subdir")
	require.NoError(t, os.Mkdir(child, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(parent, OpsFileName), []byte(""), 0o644))

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(child)

	got, err := FindOpsfileDir()
	require.NoError(t, err)
	assert.Equal(t, parent, got)
}

func TestFindOpsfileDir_NotFound(t *testing.T) {
	tmp := realPath(t, t.TempDir())

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(tmp)

	_, err := FindOpsfileDir()
	require.Error(t, err)
	assert.ErrorContains(t, err, "could not find Opsfile")
}

func TestFindOpsfileDir_DirectoryNamedOpsfileSkipped(t *testing.T) {
	parent := realPath(t, t.TempDir())
	child := filepath.Join(parent, "subdir")
	require.NoError(t, os.Mkdir(child, 0o755))
	// Create a directory named "Opsfile" in child — should be skipped.
	require.NoError(t, os.Mkdir(filepath.Join(child, OpsFileName), 0o755))
	// Place the real Opsfile in parent.
	require.NoError(t, os.WriteFile(filepath.Join(parent, OpsFileName), []byte(""), 0o644))

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(child)

	got, err := FindOpsfileDir()
	require.NoError(t, err)
	assert.Equal(t, parent, got, "directory named Opsfile should be skipped")
}
