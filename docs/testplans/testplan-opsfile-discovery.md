# Test Plan: Opsfile Discovery

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Opsfile in current directory | Run `ops` in a directory containing an `Opsfile` | Discovers and uses the Opsfile in cwd | integration |
| Opsfile in parent directory | Run `ops` in a subdirectory; Opsfile exists two levels up | Walks up and discovers the Opsfile in the ancestor | integration |
| Opsfile in root-adjacent directory | Run `ops` with Opsfile in the immediate child of `/` | Discovery walks up and finds it | integration |
| `-D` flag bypasses discovery | Run `ops -D /some/path prod cmd` with Opsfile at `/some/path/Opsfile` | Uses the specified directory, no walk-up | integration |
| `--directory` long form | Run `ops --directory /some/path prod cmd` | Same as `-D` | integration |
| Directory named Opsfile is skipped | A directory named `Opsfile` exists in cwd, but a real Opsfile exists in the parent | Discovery skips the directory and finds the file in the parent | integration |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| No Opsfile in any ancestor | Run `ops` in a temp directory with no Opsfile anywhere up to root | Error: "could not find Opsfile in any parent directory" | integration |
| `-D` points to directory without Opsfile | Run `ops -D /tmp/empty prod cmd` | Parse error from `ParseOpsFile` (file not found) | integration |
| `-D` points to nonexistent directory | Run `ops -D /nonexistent prod cmd` | Error opening file | integration |
| Opsfile is a directory at every level | Every ancestor has a directory named `Opsfile` but no file | Error: "could not find Opsfile in any parent directory" | integration |

## Existing Automated Tests

There are **no automated tests** for `getClosestOpsfilePath()`. The function is in `cmd/ops/main.go` and is not exported, and there are no `_test.go` files in `cmd/ops/`.

## Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| `-D` flag short-circuits discovery | integration | When `flags.Directory` is set, `getClosestOpsfilePath` is not called |
| `-D` with invalid path | integration | Error is surfaced when the directory does not exist or has no Opsfile |

Note: Testing `getClosestOpsfilePath` requires either extracting it to an exported function or creating `cmd/ops/main_test.go` with test helpers that set up temp directory trees.
