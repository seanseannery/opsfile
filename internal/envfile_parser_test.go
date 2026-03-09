package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTempEnvFile writes content to a temp .env file and returns its path.
func writeTempEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
	return path
}

func TestParseEnvFile_HappyPath(t *testing.T) {
	path := writeTempEnvFile(t, `
AWS_TOKEN=abc123
DB_HOST=localhost
`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, OpsVariables{
		"AWS_TOKEN": "abc123",
		"DB_HOST":   "localhost",
	}, vars)
}

func TestParseEnvFile_DoubleQuoted(t *testing.T) {
	path := writeTempEnvFile(t, `PASSWORD="my secret"`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, "my secret", vars["PASSWORD"])
}

func TestParseEnvFile_SingleQuoted(t *testing.T) {
	path := writeTempEnvFile(t, `API_KEY='sk-abc123'`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, "sk-abc123", vars["API_KEY"])
}

func TestParseEnvFile_CommentsSkipped(t *testing.T) {
	path := writeTempEnvFile(t, `# this is a comment
VAR=value
  # indented comment
`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, OpsVariables{"VAR": "value"}, vars)
}

func TestParseEnvFile_BlankLinesSkipped(t *testing.T) {
	path := writeTempEnvFile(t, `
A=1

B=2

`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, OpsVariables{"A": "1", "B": "2"}, vars)
}

func TestParseEnvFile_EnvScopedKeys(t *testing.T) {
	path := writeTempEnvFile(t, `prod_DB_PASSWORD=secret
staging_API_KEY=key123
`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, "secret", vars["prod_DB_PASSWORD"])
	assert.Equal(t, "key123", vars["staging_API_KEY"])
}

func TestParseEnvFile_EmptyNameError(t *testing.T) {
	path := writeTempEnvFile(t, `=value`)
	_, err := ParseEnvFile(path)
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing name")
	assert.ErrorContains(t, err, "line 1")
}

func TestParseEnvFile_FileNotFound(t *testing.T) {
	_, err := ParseEnvFile("/nonexistent/path/.env")
	require.Error(t, err)
	assert.ErrorContains(t, err, "env-file")
	assert.ErrorContains(t, err, "/nonexistent/path/.env")
}

func TestParseEnvFile_EmptyFile(t *testing.T) {
	path := writeTempEnvFile(t, "")
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Empty(t, vars)
}

func TestParseEnvFile_InlineComment(t *testing.T) {
	path := writeTempEnvFile(t, `VAR=value # this is a comment`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, "value", vars["VAR"])
}

func TestParseEnvFile_HashInQuotedValue(t *testing.T) {
	path := writeTempEnvFile(t, `VAR="value#with#hashes"`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, "value#with#hashes", vars["VAR"])
}

func TestParseEnvFile_EmptyValue(t *testing.T) {
	path := writeTempEnvFile(t, `VAR=`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, "", vars["VAR"])
}

func TestParseEnvFile_UnclosedQuote(t *testing.T) {
	path := writeTempEnvFile(t, `VAR="unclosed`)
	_, err := ParseEnvFile(path)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unclosed")
}

func TestParseEnvFile_LineWithoutEquals(t *testing.T) {
	// Lines without = are silently skipped (not an error)
	path := writeTempEnvFile(t, `NOEQUALS
VAR=value`)
	vars, err := ParseEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, OpsVariables{"VAR": "value"}, vars)
}
