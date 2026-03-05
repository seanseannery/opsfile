package internal

import (
	"strings"
	"testing"
)

// parseFixture is a test helper that parses an inline Opsfile string and
// returns the variables and commands maps. Fails the test on parse error.
func parseFixture(t *testing.T, content string) (OpsVariables, map[string]OpsCommand) {
	t.Helper()
	vars, commands, err := ParseOpsFile(writeTempOpsfile(t, content))
	if err != nil {
		t.Fatalf("ParseOpsFile: %v", err)
	}
	return vars, commands
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
	got, err := Resolve("list-instance-ips", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{`aws ec2 --list-instances`, `echo "done"`}
	if len(got.Lines) != len(want) {
		t.Fatalf("got %d lines, want %d: %v", len(got.Lines), len(want), got.Lines)
	}
	for i, w := range want {
		if got.Lines[i] != w {
			t.Errorf("line %d: got %q, want %q", i, got.Lines[i], w)
		}
	}
}

func TestResolve_EmptyCommandsMap(t *testing.T) {
	commands := map[string]OpsCommand{}
	vars := OpsVariables{}
	_, err := Resolve("anything", "prod", commands, vars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not contain 'not found'", err.Error())
	}
}

func TestResolve_CommandWithEmptyShellLines(t *testing.T) {
	commands := map[string]OpsCommand{
		"empty": {Name: "empty", Environments: map[string][]string{"prod": {}}},
	}
	vars := OpsVariables{}
	got, err := Resolve("empty", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Lines) != 0 {
		t.Errorf("got %d lines, want 0", len(got.Lines))
	}
}

func TestResolve_MultipleVariablesInOneLine(t *testing.T) {
	vars, commands := parseFixture(t, `
A=hello
B=world

my-cmd:
    default:
        echo $(A) $(B)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "echo hello world"
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
}

func TestResolve_VariableUsedMultipleTimes(t *testing.T) {
	vars, commands := parseFixture(t, `
A=val

my-cmd:
    default:
        echo $(A) and $(A)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "echo val and val"
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
}

func TestResolve_UnclosedDollarParen(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(incomplete"},
		}},
	}
	vars := OpsVariables{}
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "echo $(incomplete"
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
}

func TestResolve_MixedIdentifierAndNonIdentifier(t *testing.T) {
	vars, commands := parseFixture(t, `
VAR=hello

my-cmd:
    default:
        $(VAR) $(shell cmd)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "hello $(shell cmd)"
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
}

func TestResolve_DefaultFallbackVariableScoping(t *testing.T) {
	vars, commands := parseFixture(t, `
prod_ACCOUNT=prod-acct
ACCOUNT=default-acct

my-cmd:
    default:
        echo $(ACCOUNT)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "echo prod-acct"
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
}

func TestResolve_SameVarReferencedTwice(t *testing.T) {
	vars, commands := parseFixture(t, `
VAR=x

my-cmd:
    default:
        $(VAR) $(VAR)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "x x" {
		t.Errorf("got %q, want %q", got.Lines[0], "x x")
	}
}

func TestResolve_UnclosedDollarParenAtEnd(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $("},
		}},
	}
	vars := OpsVariables{}
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "echo $("
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
}

func TestResolve_EmptyToken(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $()"},
		}},
	}
	vars := OpsVariables{}
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "echo $()"
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
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
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo prod.example.com" {
		t.Errorf("got %q, want scoped value", got.Lines[0])
	}
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
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo default.example.com" {
		t.Errorf("got %q, want unscoped fallback", got.Lines[0])
	}
}

func TestResolve_MissingVariableReturnsError(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(NOPE)"},
		}},
	}
	vars := OpsVariables{}
	_, err := Resolve("my-cmd", "prod", commands, vars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not defined") {
		t.Errorf("error %q does not contain 'not defined'", err.Error())
	}
}

func TestResolve_DefaultFallback(t *testing.T) {
	vars, commands := parseFixture(t, `
prod_AWS_ACCOUNT=1234567

tail-logs:
    default:
        aws cloudwatch logs --tail $(AWS_ACCOUNT)
`)
	got, err := Resolve("tail-logs", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Lines) != 1 {
		t.Fatalf("got %d lines, want 1: %v", len(got.Lines), got.Lines)
	}
	want := "aws cloudwatch logs --tail 1234567"
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
}

func TestResolve_LocalOverridesDefault(t *testing.T) {
	vars, commands := parseFixture(t, `
tail-logs:
    default:
        aws cloudwatch logs --tail something
    local:
        docker logs myapp --follow
`)
	got, err := Resolve("tail-logs", "local", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Lines) != 1 {
		t.Fatalf("got %d lines, want 1: %v", len(got.Lines), got.Lines)
	}
	if got.Lines[0] != "docker logs myapp --follow" {
		t.Errorf("got %q, want local block line", got.Lines[0])
	}
}

func TestResolve_ScopedPriority(t *testing.T) {
	vars, commands := parseFixture(t, `
prod_AWS_ACCOUNT=scoped
AWS_ACCOUNT=unscoped

tail-logs:
    default:
        echo $(AWS_ACCOUNT)
`)
	got, err := Resolve("tail-logs", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo scoped" {
		t.Errorf("got %q, expected scoped value", got.Lines[0])
	}
}

func TestResolve_UnscopedFallback(t *testing.T) {
	vars, commands := parseFixture(t, `
AWS_ACCOUNT=unscoped

tail-logs:
    default:
        echo $(AWS_ACCOUNT)
`)
	got, err := Resolve("tail-logs", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo unscoped" {
		t.Errorf("got %q, expected unscoped value", got.Lines[0])
	}
}

func TestResolve_CommandNotFound(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        echo hello
`)
	_, err := Resolve("nonexistent", "prod", commands, vars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not contain 'not found'", err.Error())
	}
}

func TestResolve_EnvNotFoundNoDefault(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        echo hello
`)
	_, err := Resolve("my-cmd", "staging", commands, vars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no default") {
		t.Errorf("error %q does not contain 'no default'", err.Error())
	}
}

func TestResolve_VariableNotDefined(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        echo $(MISSING_VAR)
`)
	_, err := Resolve("my-cmd", "prod", commands, vars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not defined") {
		t.Errorf("error %q does not contain 'not defined'", err.Error())
	}
}

func TestResolve_NonIdentifierPassthrough(t *testing.T) {
	vars, commands := parseFixture(t, `
my-cmd:
    prod:
        echo $(aws ec2 describe-instances)
`)
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "echo $(aws ec2 describe-instances)"
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
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
	got, err := Resolve("deploy", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		"aws ecs describe-clusters --cluster my-cluster --region us-east-1",
		`echo "done in us-east-1"`,
	}
	if len(got.Lines) != len(want) {
		t.Fatalf("got %d lines, want %d: %v", len(got.Lines), len(want), got.Lines)
	}
	for i, w := range want {
		if got.Lines[i] != w {
			t.Errorf("line %d: got %q, want %q", i, got.Lines[i], w)
		}
	}
}

// --- Shell environment variable injection tests ---

func TestResolveVar_UnscopedShellEnvFallback(t *testing.T) {
	t.Setenv("VAR", "from-shell")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo from-shell" {
		t.Errorf("got %q, want shell unscoped value", got.Lines[0])
	}
}

func TestResolveVar_EnvScopedShellEnv(t *testing.T) {
	t.Setenv("prod_VAR", "shell-scoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo shell-scoped" {
		t.Errorf("got %q, want shell env-scoped value", got.Lines[0])
	}
}

func TestResolveVar_OpsfileEnvScopedBeatsShellEnvScoped(t *testing.T) {
	t.Setenv("prod_VAR", "shell-scoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	vars := OpsVariables{"prod_VAR": "opsfile-scoped"}
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo opsfile-scoped" {
		t.Errorf("got %q, want Opsfile env-scoped value", got.Lines[0])
	}
}

func TestResolveVar_ShellEnvScopedBeatsOpsfileUnscoped(t *testing.T) {
	t.Setenv("prod_VAR", "shell-scoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	vars := OpsVariables{"VAR": "opsfile-unscoped"}
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo shell-scoped" {
		t.Errorf("got %q, want shell env-scoped over Opsfile unscoped", got.Lines[0])
	}
}

func TestResolveVar_OpsfileUnscopedBeatsShellEnvUnscoped(t *testing.T) {
	t.Setenv("VAR", "shell-unscoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	vars := OpsVariables{"VAR": "opsfile-unscoped"}
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo opsfile-unscoped" {
		t.Errorf("got %q, want Opsfile unscoped over shell unscoped", got.Lines[0])
	}
}

func TestResolveVar_ShellEnvUnscopedIsLastResort(t *testing.T) {
	t.Setenv("VAR", "shell-unscoped")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo shell-unscoped" {
		t.Errorf("got %q, want shell unscoped as last resort", got.Lines[0])
	}
}

func TestResolveVar_PriorityChain(t *testing.T) {
	tests := []struct {
		name        string
		opsfileVars OpsVariables
		envVars     map[string]string // shell env vars to set
		want        string
	}{
		{
			name:        "level1 opsfile env-scoped wins",
			opsfileVars: OpsVariables{"prod_VAR": "L1", "VAR": "L3"},
			envVars:     map[string]string{"prod_VAR": "L2", "VAR": "L4"},
			want:        "L1",
		},
		{
			name:        "level2 shell env-scoped wins",
			opsfileVars: OpsVariables{"VAR": "L3"},
			envVars:     map[string]string{"prod_VAR": "L2", "VAR": "L4"},
			want:        "L2",
		},
		{
			name:        "level3 opsfile unscoped wins",
			opsfileVars: OpsVariables{"VAR": "L3"},
			envVars:     map[string]string{"VAR": "L4"},
			want:        "L3",
		},
		{
			name:        "level4 shell unscoped wins",
			opsfileVars: OpsVariables{},
			envVars:     map[string]string{"VAR": "L4"},
			want:        "L4",
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
			got, err := Resolve("my-cmd", "prod", commands, tc.opsfileVars)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := "echo " + tc.want
			if got.Lines[0] != want {
				t.Errorf("got %q, want %q", got.Lines[0], want)
			}
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
	got, err := Resolve("my-cmd", "prod", commands, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo from-opsfile from-shell" {
		t.Errorf("got %q, want mixed source substitution", got.Lines[0])
	}
}

func TestResolveVar_EmptyShellEnvValue(t *testing.T) {
	t.Setenv("VAR", "")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(VAR)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Lines[0] != "echo " {
		t.Errorf("got %q, want empty string substitution", got.Lines[0])
	}
}

func TestResolveVar_NonIdentifierUnaffectedByShellEnv(t *testing.T) {
	t.Setenv("aws", "should-not-matter")
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(aws ec2 describe-instances)"},
		}},
	}
	got, err := Resolve("my-cmd", "prod", commands, OpsVariables{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "echo $(aws ec2 describe-instances)"
	if got.Lines[0] != want {
		t.Errorf("got %q, want %q", got.Lines[0], want)
	}
}

func TestResolveVar_AbsentFromAllSources(t *testing.T) {
	commands := map[string]OpsCommand{
		"my-cmd": {Name: "my-cmd", Environments: map[string][]string{
			"prod": {"echo $(DEFINITELY_NOT_SET_XYZ123)"},
		}},
	}
	_, err := Resolve("my-cmd", "prod", commands, OpsVariables{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not defined") {
		t.Errorf("error %q does not contain 'not defined'", err.Error())
	}
}
