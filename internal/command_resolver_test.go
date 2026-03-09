package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseFixture is a test helper that parses an inline Opsfile string and
// returns the variables and commands maps. Fails the test on parse error.
func parseFixture(t *testing.T, content string) (OpsVariables, map[string]OpsCommand) {
	t.Helper()
	vars, commands, _, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err, "ParseOpsFile")
	return vars, commands
}

// lineTexts extracts the Text field from each ResolvedLine for easy comparison.
func lineTexts(rc ResolvedCommand) []string {
	out := make([]string, len(rc.Lines))
	for i, l := range rc.Lines {
		out[i] = l.Text
	}
	return out
}

func TestResolve_ExactEnvMatch(t *testing.T) {
	vars, commands := parseFixture(t, `
list-instance-ips:
    prod:
        aws ec2 --list-instances
        echo "done"
    preprod:
        aws ecs --list-instances
`)
	got, err := Resolve("list-instance-ips", "prod", commands, vars, nil)
	require.NoError(t, err)
	want := []string{`aws ec2 --list-instances`, `echo "done"`}
	assert.Equal(t, want, lineTexts(got))
}

func TestResolve_EmptyCommandsMap(t *testing.T) {
	commands := map[string]OpsCommand{}
	vars := OpsVariables{}
	_, err := Resolve("anything", "prod", commands, vars, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not found")
}

func TestResolve_CommandWithEmptyShellLines(t *testing.T) {
	commands := map[string]OpsCommand{
		"empty": {Name: "empty", Environments: map[string][]string{"prod": {}}},
	}
	vars := OpsVariables{}
	got, err := Resolve("empty", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Empty(t, got.Lines)
}

func TestResolve_MultipleVariablesInOneLine(t *testing.T) {
	vars, commands := parseFixture(t, `
A=hello
B=world

my-cmd:
    default:
        echo $(A) $(B)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo hello world", got.Lines[0].Text)
}

func TestResolve_VariableUsedMultipleTimes(t *testing.T) {
	vars, commands := parseFixture(t, `
A=val

my-cmd:
    default:
        echo $(A) and $(A)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo val and val", got.Lines[0].Text)
}

func TestResolve_UnclosedDollarParen(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(incomplete"},
		}},
	}
	vars := OpsVariables{}
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo $(incomplete", got.Lines[0].Text)
}

func TestResolve_MixedIdentifierAndNonIdentifier(t *testing.T) {
	vars, commands := parseFixture(t, `
VAR=hello

my-cmd:
    default:
        $(VAR) $(shell cmd)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello $(shell cmd)", got.Lines[0].Text)
}

func TestResolve_DefaultFallbackVariableScoping(t *testing.T) {
	vars, commands := parseFixture(t, `
prod_ACCOUNT=prod-acct
ACCOUNT=default-acct

my-cmd:
    default:
        echo $(ACCOUNT)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo prod-acct", got.Lines[0].Text)
}

func TestResolve_SameVarReferencedTwice(t *testing.T) {
	vars, commands := parseFixture(t, `
VAR=x

my-cmd:
    default:
        $(VAR) $(VAR)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "x x", got.Lines[0].Text)
}

func TestResolve_UnclosedDollarParenAtEnd(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $("},
		}},
	}
	vars := OpsVariables{}
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo $(", got.Lines[0].Text)
}

func TestResolve_EmptyToken(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $()"},
		}},
	}
	vars := OpsVariables{}
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo $()", got.Lines[0].Text)
}

func TestResolve_ScopedLookupPriority(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(HOST)"},
		}},
	}
	vars := OpsVariables{
		"prod_HOST": "prod.example.com",
		"HOST":      "default.example.com",
	}
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo prod.example.com", got.Lines[0].Text)
}

func TestResolve_UnscopedFallbackDirect(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(HOST)"},
		}},
	}
	vars := OpsVariables{
		"HOST": "default.example.com",
	}
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo default.example.com", got.Lines[0].Text)
}

func TestResolve_MissingVariableReturnsError(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(NOPE)"},
		}},
	}
	vars := OpsVariables{}
	_, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not defined")
}

func TestResolve_DefaultFallback(t *testing.T) {
	vars, commands := parseFixture(t, `
prod_AWS_ACCOUNT=1234567

tail-logs:
    default:
        aws cloudwatch logs --tail $(AWS_ACCOUNT)
`)
	got, err := Resolve("tail-logs", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "aws cloudwatch logs --tail 1234567", got.Lines[0].Text)
}

func TestResolve_LocalOverridesDefault(t *testing.T) {
	vars, commands := parseFixture(t, `
tail-logs:
    default:
        aws cloudwatch logs --tail something
    local:
        docker logs myapp --follow
`)
	got, err := Resolve("tail-logs", "local", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "docker logs myapp --follow", got.Lines[0].Text)
}

func TestResolve_ScopedPriority(t *testing.T) {
	vars, commands := parseFixture(t, `
prod_AWS_ACCOUNT=scoped
AWS_ACCOUNT=unscoped

tail-logs:
    default:
        echo $(AWS_ACCOUNT)
`)
	got, err := Resolve("tail-logs", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo scoped", got.Lines[0].Text)
}

func TestResolve_UnscopedFallback(t *testing.T) {
	vars, commands := parseFixture(t, `
AWS_ACCOUNT=unscoped

tail-logs:
    default:
        echo $(AWS_ACCOUNT)
`)
	got, err := Resolve("tail-logs", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo unscoped", got.Lines[0].Text)
}

func TestResolve_CommandNotFound(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        echo hello
`)
	_, err := Resolve("nonexistent", "prod", commands, vars, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not found")
}

func TestResolve_EnvNotFoundNoDefault(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        echo hello
`)
	_, err := Resolve("my-cmd", "staging", commands, vars, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "no default")
}

func TestResolve_VariableNotDefined(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        echo $(MISSING_VAR)
`)
	_, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not defined")
}

func TestResolve_NonIdentifierPassthrough(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        echo $(aws ec2 describe-instances)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo $(aws ec2 describe-instances)", got.Lines[0].Text)
}

func TestResolve_MultiLineCommand(t *testing.T) {
	vars, commands := parseFixture(t, `
REGION=us-east-1
prod_CLUSTER=my-cluster

deploy:
    prod:
        aws ecs describe-clusters --cluster $(CLUSTER) --region $(REGION)
        echo "done in $(REGION)"
`)
	got, err := Resolve("deploy", "prod", commands, vars, nil)
	require.NoError(t, err)
	want := []string{
		"aws ecs describe-clusters --cluster my-cluster --region us-east-1",
		`echo "done in us-east-1"`,
	}
	assert.Equal(t, want, lineTexts(got))
}

// --- Shell environment variable injection tests ---

func TestResolveVar_UnscopedShellEnvFallback(t *testing.T) {
	t.Setenv("VAR", "from-shell")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo from-shell", got.Lines[0].Text)
}

func TestResolveVar_EnvScopedShellEnv(t *testing.T) {
	t.Setenv("prod_VAR", "shell-scoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo shell-scoped", got.Lines[0].Text)
}

func TestResolveVar_ShellEnvScopedBeatsOpsfileEnvScoped(t *testing.T) {
	t.Setenv("prod_VAR", "shell-scoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	vars := OpsVariables{"prod_VAR": "opsfile-scoped"}
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo shell-scoped", got.Lines[0].Text)
}

func TestResolveVar_ShellEnvScopedBeatsOpsfileUnscoped(t *testing.T) {
	t.Setenv("prod_VAR", "shell-scoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	vars := OpsVariables{"VAR": "opsfile-unscoped"}
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo shell-scoped", got.Lines[0].Text)
}

func TestResolveVar_ShellUnscopedBeatsOpsfileUnscoped(t *testing.T) {
	t.Setenv("VAR", "shell-unscoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	vars := OpsVariables{"VAR": "opsfile-unscoped"}
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo shell-unscoped", got.Lines[0].Text)
}

func TestResolveVar_ShellUnscopedIsPriority4(t *testing.T) {
	t.Setenv("VAR", "shell-unscoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo shell-unscoped", got.Lines[0].Text)
}

func TestResolveVar_PriorityChain(t *testing.T) {
	tests := []struct {
		name        string
		opsfileVars OpsVariables
		envFileVars OpsVariables
		envVars     map[string]string // shell env vars to set
		want        string
	}{
		{
			name:        "p1 shell env-scoped wins",
			opsfileVars: OpsVariables{"prod_VAR": "opsfile-scoped", "VAR": "opsfile-unscoped"},
			envFileVars: OpsVariables{"prod_VAR": "envfile-scoped", "VAR": "envfile-unscoped"},
			envVars:     map[string]string{"prod_VAR": "shell-scoped", "VAR": "shell-unscoped"},
			want:        "shell-scoped",
		},
		{
			name:        "p2 opsfile env-scoped wins",
			opsfileVars: OpsVariables{"prod_VAR": "opsfile-scoped", "VAR": "opsfile-unscoped"},
			envFileVars: OpsVariables{"prod_VAR": "envfile-scoped", "VAR": "envfile-unscoped"},
			envVars:     map[string]string{"VAR": "shell-unscoped"},
			want:        "opsfile-scoped",
		},
		{
			name:        "p3 env-file env-scoped wins",
			opsfileVars: OpsVariables{"VAR": "opsfile-unscoped"},
			envFileVars: OpsVariables{"prod_VAR": "envfile-scoped", "VAR": "envfile-unscoped"},
			envVars:     map[string]string{"VAR": "shell-unscoped"},
			want:        "envfile-scoped",
		},
		{
			name:        "p4 shell unscoped wins",
			opsfileVars: OpsVariables{"VAR": "opsfile-unscoped"},
			envFileVars: OpsVariables{"VAR": "envfile-unscoped"},
			envVars:     map[string]string{"VAR": "shell-unscoped"},
			want:        "shell-unscoped",
		},
		{
			name:        "p5 opsfile unscoped wins",
			opsfileVars: OpsVariables{"VAR": "opsfile-unscoped"},
			envFileVars: OpsVariables{"VAR": "envfile-unscoped"},
			envVars:     map[string]string{},
			want:        "opsfile-unscoped",
		},
		{
			name:        "p6 env-file unscoped wins",
			opsfileVars: OpsVariables{},
			envFileVars: OpsVariables{"VAR": "envfile-unscoped"},
			envVars:     map[string]string{},
			want:        "envfile-unscoped",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			commands := map[string]OpsCommand{
				"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
					"prod": {"echo $(VAR)"},
				}},
			}
			got, err := Resolve("my-cmd", "prod", commands, tc.opsfileVars, tc.envFileVars)
			require.NoError(t, err)
			assert.Equal(t, "echo "+tc.want, got.Lines[0].Text)
		})
	}
}

func TestResolveVar_MixedSources(t *testing.T) {
	t.Setenv("B", "from-shell")
	vars := OpsVariables{"A": "from-opsfile"}
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(A) $(B)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo from-opsfile from-shell", got.Lines[0].Text)
}

func TestResolveVar_EmptyShellEnvValue(t *testing.T) {
	t.Setenv("VAR", "")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo ", got.Lines[0].Text)
}

func TestResolveVar_NonIdentifierUnaffectedByShellEnv(t *testing.T) {
	t.Setenv("aws", "should-not-matter")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(aws ec2 describe-instances)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo $(aws ec2 describe-instances)", got.Lines[0].Text)
}

func TestResolveVar_AbsentFromAllSources(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(DEFINITELY_NOT_SET_XYZ123)"},
		}},
	}
	_, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not defined")
}

func TestResolveVar_EnvFileScopedBeatsOpsfileUnscoped(t *testing.T) {
	// P3 (env-file env-scoped) must beat P5 (Opsfile unscoped).
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	vars := OpsVariables{"VAR": "opsfile-unscoped"} // P5
	envFileVars := OpsVariables{"prod_VAR": "envfile-scoped"} // P3
	got, err := Resolve("my-cmd", "prod", commands, vars, envFileVars)
	require.NoError(t, err)
	assert.Equal(t, "echo envfile-scoped", got.Lines[0].Text)
}

func TestResolveVar_EnvFileScopedKeyWrongEnv(t *testing.T) {
	// An env-file key scoped to a different env must not resolve for the current env.
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	envFileVars := OpsVariables{"staging_VAR": "wrong-env-value"}
	_, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, envFileVars)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not defined")
}

// --- @ prefix tests ---

func TestResolve_AtPrefixStripped(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"@echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_NoAtPrefix(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo hello", got.Lines[0].Text)
	assert.False(t, got.Lines[0].Silent)
}

func TestResolve_MixedAtAndNonAt(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"@echo setup", "aws deploy", "@echo cleanup"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 3)
	assert.Equal(t, "echo setup", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
	assert.Equal(t, "aws deploy", got.Lines[1].Text)
	assert.False(t, got.Lines[1].Silent)
	assert.Equal(t, "echo cleanup", got.Lines[2].Text)
	assert.True(t, got.Lines[2].Silent)
}

func TestResolve_AtPrefixWithVariable(t *testing.T) {
	vars, commands := parseFixture(t, `
VAR=hello

my-cmd:
    default:
        @echo $(VAR)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_AtPrefixWithScopedVariable(t *testing.T) {
	vars, commands := parseFixture(t, `
prod_ACCT=123

my-cmd:
    default:
        @echo $(ACCT)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo 123", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_AtPrefixWithBackslashContinuation(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        @aws logs \
            --follow
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "aws logs --follow", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_AtPrefixWithIndentContinuation(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        @aws logs
            --follow
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "aws logs --follow", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_DoubleAtPrefix(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"@@echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "@echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_AtPrefixOnly(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"@"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_AtInMiddleOfLine(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo user@host.com"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo user@host.com", got.Lines[0].Text)
	assert.False(t, got.Lines[0].Silent)
}

func TestResolve_AtInVariableValue(t *testing.T) {
	vars, commands := parseFixture(t, `
VAR=user@host

my-cmd:
    default:
        echo $(VAR)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo user@host", got.Lines[0].Text)
	assert.False(t, got.Lines[0].Silent)
}

func TestResolve_AtPrefixNonIdentifierPassthrough(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"@$(shell cmd)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "$(shell cmd)", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_AtPrefixMissingVariable(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"@echo $(MISSING)"},
		}},
	}
	_, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not defined")
}

func TestResolve_MultiLineMixedAt(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        @echo setup
        echo deploy
        @echo cleanup
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 3)

	wantTexts := []string{"echo setup", "echo deploy", "echo cleanup"}
	wantSilent := []bool{true, false, true}
	assert.Equal(t, wantTexts, lineTexts(got))
	for i, line := range got.Lines {
		assert.Equal(t, wantSilent[i], line.Silent, "line %d Silent", i)
	}
}

func TestResolve_AtPrefixBackslashTrailingEOF(t *testing.T) {
	content := `
my-cmd:
    prod:
        @aws logs \`
	_, commands, _, _, err := ParseOpsFile(writeTempOpsfile(t, content))
	require.NoError(t, err)
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "aws logs ", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_AtOnContinuationFragment(t *testing.T) {
	// @ on a non-first continuation fragment is part of the joined shell text,
	// not Opsfile syntax (per FR-5).
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        aws logs \
            @--follow
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "aws logs @--follow", got.Lines[0].Text)
	assert.False(t, got.Lines[0].Silent)
}

func TestResolve_AtPrefixWhitespaceOnlyAfter(t *testing.T) {
	// @ followed by only whitespace — silent flag set, text is the whitespace.
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"@   "},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "   ", got.Lines[0].Text)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_ExistingTestsNoSilent(t *testing.T) {
	// Verify that existing commands without @ have Silent=false
	vars, commands := parseFixture(t, `
REGION=us-east-1

deploy:
    prod:
        aws ecs update --region $(REGION)
        echo done
`)
	got, err := Resolve("deploy", "prod", commands, vars, nil)
	require.NoError(t, err)
	for i, line := range got.Lines {
		assert.False(t, line.Silent, "line %d should not be silent", i)
	}
}

// --- - (dash) prefix tests ---

func TestResolve_DashPrefixStripped(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"-echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
	assert.False(t, got.Lines[0].Silent)
}

func TestResolve_NoDashPrefix(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo hello", got.Lines[0].Text)
	assert.False(t, got.Lines[0].IgnoreError)
	assert.False(t, got.Lines[0].Silent)
}

func TestResolve_DashAtPrefix(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"-@echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_AtDashPrefix(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"@-echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_DashPrefixWithVariable(t *testing.T) {
	vars, commands := parseFixture(t, `
VAR=hello

my-cmd:
    default:
        -echo $(VAR)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
}

func TestResolve_DashPrefixWithScopedVariable(t *testing.T) {
	vars, commands := parseFixture(t, `
prod_ACCT=123

my-cmd:
    default:
        -echo $(ACCT)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "echo 123", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
}

func TestResolve_DashPrefixWithBackslashContinuation(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        -aws logs \
            --follow
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "aws logs --follow", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
}

func TestResolve_DashPrefixWithIndentContinuation(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        -aws logs
            --follow
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "aws logs --follow", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
}

func TestResolve_MixedDashAndNonDash(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"-docker stop app", "docker run app"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 2)
	assert.Equal(t, "docker stop app", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
	assert.Equal(t, "docker run app", got.Lines[1].Text)
	assert.False(t, got.Lines[1].IgnoreError)
}

func TestResolve_MultiLineMixedDashAt(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        -@echo setup
        echo deploy
        -echo cleanup
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 3)

	wantTexts := []string{"echo setup", "echo deploy", "echo cleanup"}
	wantSilent := []bool{true, false, false}
	wantIgnoreError := []bool{true, false, true}
	assert.Equal(t, wantTexts, lineTexts(got))
	for i, line := range got.Lines {
		assert.Equal(t, wantSilent[i], line.Silent, "line %d Silent", i)
		assert.Equal(t, wantIgnoreError[i], line.IgnoreError, "line %d IgnoreError", i)
	}
}

func TestResolve_DoubleDashPrefix(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"--echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "-echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
}

func TestResolve_DashPrefixOnly(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"-"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
	assert.False(t, got.Lines[0].Silent)
}

func TestResolve_DashAtPrefixOnly(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"-@"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_DashInMiddleOfLine(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"kubectl delete --force"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "kubectl delete --force", got.Lines[0].Text)
	assert.False(t, got.Lines[0].IgnoreError)
}

func TestResolve_DashInVariableValue(t *testing.T) {
	vars, commands := parseFixture(t, `
VAR=hello-world

my-cmd:
    default:
        echo $(VAR)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "echo hello-world", got.Lines[0].Text)
	assert.False(t, got.Lines[0].IgnoreError)
}

func TestResolve_DashOnContinuationFragment(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        aws logs \
            -follow
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "aws logs -follow", got.Lines[0].Text)
	assert.False(t, got.Lines[0].IgnoreError)
}

func TestResolve_DashPrefixMissingVariable(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"-echo $(MISSING)"},
		}},
	}
	_, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not defined")
}

func TestResolve_DashPrefixWhitespaceAfter(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"-   "},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "   ", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
}

func TestResolve_DashAtWithBackslashContinuation(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        -@aws stop \
            --force
`)
	got, err := Resolve("my-cmd", "prod", commands, vars, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "aws stop --force", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_DashAtDashPrefix(t *testing.T) {
	// -@- should strip one - and one @, leaving - as shell text
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"-@-echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "-echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
	assert.True(t, got.Lines[0].Silent)
}

func TestResolve_AtDashAtPrefix(t *testing.T) {
	// @-@ should strip one @ and one -, leaving @ as shell text
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"@-@echo hello"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{}, nil)
	require.NoError(t, err)
	require.Len(t, got.Lines, 1)
	assert.Equal(t, "@echo hello", got.Lines[0].Text)
	assert.True(t, got.Lines[0].IgnoreError)
	assert.True(t, got.Lines[0].Silent)
}
