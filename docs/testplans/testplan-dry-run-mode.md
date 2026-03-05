# Test Plan: Dry-Run Mode

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Dry-run prints resolved lines | `--dry-run` flag with a valid command | Each resolved shell line printed to stdout, one per line | e2e |
| Dry-run does not execute | `--dry-run` with a command that would fail (e.g., `exit 1`) | Lines printed but no execution error | e2e |
| Variable substitution occurs in dry-run | `--dry-run` with `$(VAR)` references | Printed lines show substituted values | e2e |
| Environment selection occurs in dry-run | `--dry-run` with env-specific and default blocks | Correct environment's lines are printed | e2e |
| Multi-line command in dry-run | Command with 3 shell lines | All 3 lines printed | e2e |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Dry-run with silent suppresses output | `--dry-run --silent` | Nothing printed, nothing executed | e2e |
| Dry-run with undefined variable | `--dry-run` referencing `$(MISSING)` | Error from variable resolution (not suppressed by dry-run) | e2e |
| Dry-run with invalid command name | `--dry-run` with nonexistent command | Error from command resolution | e2e |

## Existing Automated Tests

There are **no automated tests** for dry-run mode. The dry-run logic is in `cmd/ops/main.go` (lines 63-69) and there are no test files in `cmd/ops/`.

## Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| Dry-run prints resolved lines to stdout | e2e | Build `ops`, run with `--dry-run`, capture stdout, verify lines match expected |
| Dry-run does not execute commands | e2e | Use a command that creates a side-effect (e.g., touch a file); verify the file is not created |
| Dry-run with `--silent` produces no output | e2e | Capture stdout/stderr, verify both are empty |
| Dry-run still resolves variables | e2e | Printed lines contain substituted variable values, not `$(VAR)` tokens |
| Dry-run with `-d` short flag | e2e | Short flag form works identically to `--dry-run` |
| Dry-run exits with code 0 on success | e2e | Process exit code is 0 when dry-run completes without resolution errors |

Note: Dry-run is a `main()` flow concern. Testing requires either e2e tests (build and run the binary) or refactoring `main()` to be testable.
