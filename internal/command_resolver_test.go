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
