# Test Plan: CLI Flag Parsing

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| No flags, positionals pass through | `["prod", "tail-logs"]` | `OpsFlags{}`, positionals `["prod", "tail-logs"]` | unit |
| Empty input | `[]` | `OpsFlags{}`, positionals `[]` | unit |
| `-D` sets Directory | `["-D", "/path", "prod", "cmd"]` | `Directory: "/path"` | unit |
| `--directory` sets Directory | `["--directory", "/path", "prod", "cmd"]` | `Directory: "/path"` | unit |
| `-d` sets DryRun | `["-d", "prod", "cmd"]` | `DryRun: true` | unit |
| `--dry-run` sets DryRun | `["--dry-run", "prod", "cmd"]` | `DryRun: true` | unit |
| `-s` sets Silent | `["-s", "prod", "cmd"]` | `Silent: true` | unit |
| `--silent` sets Silent | `["--silent", "prod", "cmd"]` | `Silent: true` | unit |
| `-v` sets Version | `["-v"]` | `Version: true` | unit |
| `--version` sets Version | `["--version"]` | `Version: true` | unit |
| `-h` returns ErrHelp | `["-h"]` | Returns `ErrHelp` sentinel | unit |
| `--help` returns ErrHelp | `["--help"]` | Returns `ErrHelp` sentinel | unit |
| `-?` returns ErrHelp | `["-?"]` | Returns `ErrHelp` sentinel | unit |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Unknown flag `-z` | `["-z"]` | Error containing "flag provided but not defined" | unit |
| Unknown long flag `--foobar` | `["--foobar"]` | Error containing "flag provided but not defined" | unit |
| `-D` missing value | `["-D"]` | Error (flag needs an argument) | unit |

## Existing Automated Tests

- `TestParseOpsFlags` in `internal/flag_parser_test.go` (line 8)
  - 13 subtests covering: no flags, empty input, `-D`, `--directory`, `-d`, `--dry-run`, `-s`, `--silent`, `-v`, `--version`, `-h`, `--help`, `-?`, unknown flag `-z`

## Missing Automated Tests

All previously missing tests have been implemented and pass.
