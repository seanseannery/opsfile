# Test Plan: Shell Execution

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Single successful command | `["true"]` with `/bin/sh` | No error | unit |
| Multiple successful commands | `["true", "true", "true"]` | No error, all run sequentially | unit |
| Empty lines list | `[]` | No error, no-op | unit |
| Command produces stdout | `["echo hello"]` | `hello` printed to stdout | integration |
| Shell selection from $SHELL | `$SHELL=/bin/bash`, lines use bash syntax | Commands run under bash | integration |
| Fallback to /bin/sh | `$SHELL` unset | Commands run under `/bin/sh` | integration |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Single failing command | `["false"]` | Error with exit code 1 | unit |
| Stops on first failure | `["false", "true"]` | Error from first command, second never runs | unit |
| Middle command fails | `["true", "false", "true"]` | Error from second command, third never runs | unit |
| Custom exit code propagated | `["exit 42"]` | `*exec.ExitError` with exit code 42 | unit |
| Invalid shell binary | `lines` with shell set to `/nonexistent` | Error (exec failure) | unit |

## Existing Automated Tests

### `internal/executor_test.go`

- `TestExecute` (line 9) -- table-driven test with 7 subtests:
  - `"single successful command"` -- `["true"]` succeeds
  - `"multiple successful commands"` -- `["true", "true", "true"]` succeeds
  - `"single failing command"` -- `["false"]` returns exit code 1
  - `"stops on first failure"` -- `["false", "true"]` stops at first
  - `"middle command fails"` -- `["true", "false", "true"]` stops at middle
  - `"exit code is propagated"` -- `["exit 42"]` returns exit code 42
  - `"empty lines list"` -- `[]` is a no-op

## Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| Command inherits stdin | integration | A command that reads from stdin receives input (e.g., piped input) |
| Shell with non-executable path | unit | Shell set to a file that is not executable -- returns error |
| $SHELL fallback logic in main.go | integration | When `SHELL` env var is empty, `/bin/sh` is used (this logic is in `main.go`, not `executor.go`) |
