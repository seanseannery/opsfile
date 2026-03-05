# Test Plan: CLI Argument Parsing

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Environment and command parsed | `["prod", "tail-logs"]` | `Args{OpsEnv: "prod", OpsCommand: "tail-logs"}` | unit |
| Passthrough args preserved | `["prod", "tail-logs", "--since", "1h"]` | `CommandArgs: ["--since", "1h"]` in order | unit |
| Single passthrough arg | `["prod", "cmd", "extra"]` | `CommandArgs: ["extra"]` | unit |
| Many passthrough args | `["prod", "cmd", "a", "b", "c", "d"]` | All four args preserved in order | unit |
| Hyphenated environment name | `["my-env", "my-cmd"]` | `OpsEnv: "my-env"` | unit |
| Hyphenated command name | `["prod", "tail-logs"]` | `OpsCommand: "tail-logs"` | unit |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| No arguments at all | `[]` | Error containing "missing environment" | unit |
| Only environment, no command | `["prod"]` | Error containing "missing command" | unit |

## Existing Automated Tests

- `TestParseOpsArgs` in `internal/flag_parser_test.go` (line 136)
  - `"env and command"` -- parses two positionals
  - `"env, command, and passthrough args"` -- parses with extra args
  - `"no args"` -- error for missing environment
  - `"only env"` -- error for missing command

## Missing Automated Tests

All previously missing tests have been implemented and pass.
