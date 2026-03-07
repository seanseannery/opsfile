package internal

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatCommandList(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		cmds     map[string]OpsCommand
		cmdOrder []string
		envOrder []string
		want     string
	}{
		{
			name: "commands with descriptions",
			path: "./Opsfile",
			cmds: map[string]OpsCommand{
				"tail-logs":         {Name: "tail-logs", Description: "Tail CloudWatch logs", Environments: map[string][]string{"prod": {}}},
				"show-profile":      {Name: "show-profile", Description: "Using AWS profile", Environments: map[string][]string{"default": {}}},
				"list-instance-ips": {Name: "list-instance-ips", Description: "List the private IPs of running instances", Environments: map[string][]string{"prod": {}}},
			},
			cmdOrder: []string{"show-profile", "tail-logs", "list-instance-ips"},
			envOrder: []string{"default", "local", "preprod", "prod"},
			want: "Commands Found in [./Opsfile]:\n" +
				"\n" +
				"Environments:\n" +
				"  default  local  preprod  prod\n" +
				"\n" +
				"Commands:\n" +
				"  show-profile       Using AWS profile\n" +
				"  tail-logs          Tail CloudWatch logs\n" +
				"  list-instance-ips  List the private IPs of running instances\n",
		},
		{
			name: "commands without descriptions",
			path: "./Opsfile",
			cmds: map[string]OpsCommand{
				"deploy":  {Name: "deploy", Environments: map[string][]string{"prod": {}}},
				"restart": {Name: "restart", Environments: map[string][]string{"prod": {}}},
			},
			cmdOrder: []string{"deploy", "restart"},
			envOrder: []string{"prod"},
			want: "Commands Found in [./Opsfile]:\n" +
				"\n" +
				"Environments:\n" +
				"  prod\n" +
				"\n" +
				"Commands:\n" +
				"  deploy\n" +
				"  restart\n",
		},
		{
			name: "mixed descriptions and no descriptions",
			path: "examples/Opsfile",
			cmds: map[string]OpsCommand{
				"build":  {Name: "build", Description: "Build the project", Environments: map[string][]string{"default": {}}},
				"deploy": {Name: "deploy", Environments: map[string][]string{"prod": {}}},
				"test":   {Name: "test", Description: "Run tests", Environments: map[string][]string{"default": {}}},
			},
			cmdOrder: []string{"build", "deploy", "test"},
			envOrder: []string{"default", "prod"},
			want: "Commands Found in [examples/Opsfile]:\n" +
				"\n" +
				"Environments:\n" +
				"  default  prod\n" +
				"\n" +
				"Commands:\n" +
				"  build   Build the project\n" +
				"  deploy\n" +
				"  test    Run tests\n",
		},
		{
			name: "environment order preserved",
			path: "./Opsfile",
			cmds: map[string]OpsCommand{
				"cmd": {Name: "cmd", Environments: map[string][]string{"zebra": {}, "alpha": {}}},
			},
			cmdOrder: []string{"cmd"},
			envOrder: []string{"zebra", "alpha"},
			want: "Commands Found in [./Opsfile]:\n" +
				"\n" +
				"Environments:\n" +
				"  zebra  alpha\n" +
				"\n" +
				"Commands:\n" +
				"  cmd\n",
		},
		{
			name: "column alignment with varying name lengths",
			path: "./Opsfile",
			cmds: map[string]OpsCommand{
				"a":              {Name: "a", Description: "short name", Environments: map[string][]string{"default": {}}},
				"very-long-name": {Name: "very-long-name", Description: "long name", Environments: map[string][]string{"default": {}}},
				"mid":            {Name: "mid", Description: "medium", Environments: map[string][]string{"default": {}}},
			},
			cmdOrder: []string{"a", "very-long-name", "mid"},
			envOrder: []string{"default"},
			want: "Commands Found in [./Opsfile]:\n" +
				"\n" +
				"Environments:\n" +
				"  default\n" +
				"\n" +
				"Commands:\n" +
				"  a               short name\n" +
				"  very-long-name  long name\n" +
				"  mid             medium\n",
		},
		{
			name:     "single command",
			path:     "./Opsfile",
			cmds:     map[string]OpsCommand{"solo": {Name: "solo", Description: "Only command", Environments: map[string][]string{"prod": {}}}},
			cmdOrder: []string{"solo"},
			envOrder: []string{"prod"},
			want: "Commands Found in [./Opsfile]:\n" +
				"\n" +
				"Environments:\n" +
				"  prod\n" +
				"\n" +
				"Commands:\n" +
				"  solo  Only command\n",
		},
		{
			name:     "empty command map",
			path:     "./Opsfile",
			cmds:     map[string]OpsCommand{},
			cmdOrder: []string{},
			envOrder: []string{},
			want: "Commands Found in [./Opsfile]:\n" +
				"\n" +
				"Environments:\n" +
				"  \n" +
				"\n" +
				"Commands:\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			FormatCommandList(&buf, tc.path, tc.cmds, tc.cmdOrder, tc.envOrder)
			assert.Equal(t, tc.want, buf.String())
		})
	}
}
