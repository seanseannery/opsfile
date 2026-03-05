# Test Plan: Help / Usage Output

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| `-h` prints usage and returns ErrHelp | `["-h"]` | Usage printed to stderr, `ErrHelp` returned | unit |
| `--help` prints usage and returns ErrHelp | `["--help"]` | Usage printed to stderr, `ErrHelp` returned | unit |
| `-?` prints usage and returns ErrHelp | `["-?"]` | Usage printed to stderr, `ErrHelp` returned | unit |
| Help exits with code 0 | Run `ops -h` | Process exits with code 0 | e2e |
| Usage banner contains invocation syntax | Run `ops --help` | Output contains `ops [flags] <environment> <command>` | e2e |
| Usage banner contains examples | Run `ops --help` | Output contains `ops preprod open-dashboard` | e2e |
| Usage banner lists all flags | Run `ops --help` | Output contains `-D`, `-d`, `-s`, `-v` flag descriptions | e2e |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Unknown flag triggers error, not help | `["-z"]` | Error returned (not `ErrHelp`), usage may be printed by flag package | unit |

## Existing Automated Tests

### `internal/flag_parser_test.go`

- `TestParseOpsFlags` subtests (line 8):
  - `"-h returns ErrHelp"` (line 78) -- verifies `-h` returns `ErrHelp` sentinel
  - `"--help returns ErrHelp"` (line 83) -- verifies `--help` returns `ErrHelp` sentinel
  - `"-? returns ErrHelp"` (line 88) -- verifies `-?` returns `ErrHelp` sentinel

These tests verify the sentinel error is returned but do **not** verify the content of the usage output.

## Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| Usage output contains invocation syntax | unit | Capture stderr from `ParseOpsFlags(["-h"])`, verify it contains `ops [flags] <environment> <command>` |
| Usage output contains description | unit | Stderr contains "runs commonly-used live-operation commands" |
| Usage output contains examples | unit | Stderr contains example invocations |
| Help exits with code 0 in main | e2e | Build and run `ops --help`, verify exit code is 0 |
