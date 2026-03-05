# Test Plan: Version Reporting

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| `-v` prints version string | Run `ops -v` | Output: `ops version <version> (<os>/<arch>)` | e2e |
| `--version` prints version string | Run `ops --version` | Same output as `-v` | e2e |
| Default version is 0.0.1 | Build without `-ldflags`, run `-v` | Version shows `0.0.1` | e2e |
| Build-time version override | Build with `-ldflags="-X .../internal.Version=1.2.3"`, run `-v` | Version shows `1.2.3` | e2e |
| Version exits with code 0 | Run `ops -v` | Process exits with code 0 | e2e |
| Version includes OS and arch | Run `ops -v` | Output contains `darwin/arm64` or equivalent `GOOS/GOARCH` | e2e |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Version flag with other flags | `ops -v -d` | Version printed and exits; dry-run flag ignored | e2e |
| Version flag before positionals | `ops -v prod cmd` | Version printed and exits; positionals ignored | e2e |

## Existing Automated Tests

There are **no automated tests** for version reporting. The logic is split between `internal/version.go` (variable declaration) and `cmd/ops/main.go` (output formatting and exit).

The `-v` and `--version` flags are tested in `TestParseOpsFlags` (`internal/flag_parser_test.go`, lines 66-76) -- these verify that the `Version` bool is set to `true`, but do not test the actual version output or exit behavior.

## Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| Version output format | e2e | Build binary, run with `-v`, verify output matches `ops version X.Y.Z (os/arch)` regex |
| Version exits with code 0 | e2e | Binary run with `-v` exits with code 0 |
| Version flag takes precedence over other flags | e2e | `ops -v -d prod cmd` prints version and exits without error |
| Build-time ldflags override | e2e | Build with custom version, verify output reflects the override |
