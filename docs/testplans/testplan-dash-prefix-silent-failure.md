# Test Plan: `-` Prefix to Ignore Non-Zero Exit Codes

**Feature Doc:** [docs/feat-dash-prefix-silent-failure.md](../feat-dash-prefix-silent-failure.md)

---

## 1. Scope

### In Scope
- `-` prefix detection and stripping in the command resolver
- `IgnoreError` field on `ResolvedLine` struct
- Executor error suppression when `IgnoreError` is true
- Combined `-@` and `@-` prefix handling (order-independent)
- Multi-line continuation with `-` prefix (backslash and indent)
- `--dry-run` compatibility (prints resolved lines with `-` already stripped)
- Variable substitution on lines with `-` prefix
- Regression testing for existing `@` prefix, resolver, and executor behavior

### Out of Scope
- Suppressing stderr output from the failing command
- Applying `-` to an entire command block via a single annotation
- Retry logic or conditional execution based on exit codes

---

## 2. Test Objectives

- Verify that `-` prefix is correctly stripped from resolved command lines and `IgnoreError` is set to true
- Verify that the executor ignores non-zero exit codes when `IgnoreError` is true and continues to the next line
- Verify that commands without `-` still fail-fast on non-zero exit codes (no regression)
- Verify combined `-@` and `@-` prefixes work in either order, setting both `Silent` and `IgnoreError`
- Verify `--dry-run` prints resolved lines with `-` already stripped
- Verify no regressions in existing `@` prefix behavior or executor error handling

---

## 3. Entry & Exit Criteria

### Entry Criteria (prerequisites before testing begins)
- [ ] Feature implementation is code-complete
- [ ] Code compiles without errors (`make build`)
- [ ] Existing tests still pass (`make test`)

### Exit Criteria (conditions to consider testing complete)
- [ ] All green path automated tests pass
- [ ] All red path automated tests pass
- [ ] All green/red manual tests have been run and pass
- [ ] No regressions in related features
- [ ] Test coverage does not decrease

### Acceptance Criteria

- [ ] Lines with `-` prefix have `IgnoreError: true` and `-` is stripped from the command text
- [ ] Executor continues to the next line when a `-` prefixed command returns non-zero
- [ ] Executor still returns an error for non-zero exit codes on lines without `-` prefix
- [ ] `-@` and `@-` both set `Silent: true` AND `IgnoreError: true`, with both prefix characters stripped
- [ ] `--cmd` strips one `-` and passes `-cmd` as shell text (single-layer stripping, consistent with `@@`)
- [ ] `-` in the middle of a line (e.g., `kubectl delete --force`) is not treated as Opsfile syntax
- [ ] `--dry-run` shows resolved commands without `-` prefix
- [ ] Multi-line commands with `-` on the first fragment inherit `IgnoreError` for the entire joined command
- [ ] Variable substitution works correctly on `-`-prefixed lines
- [ ] No breaking changes to existing `@` prefix behavior or fail-fast error handling

---

## 4. Green Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| Basic - strip | TestResolve_DashPrefixStripped | Opsfile line: `-echo hello` | `ResolvedLine{Text: "echo hello", IgnoreError: true, Silent: false}` | automated (unit) |
| No - prefix | TestResolve_NoDashPrefix | Opsfile line: `echo hello` | `ResolvedLine{Text: "echo hello", IgnoreError: false, Silent: false}` | automated (unit) |
| Combined -@ prefix | TestResolve_DashAtPrefix | Opsfile line: `-@echo hello` | `ResolvedLine{Text: "echo hello", IgnoreError: true, Silent: true}` | automated (unit) |
| Combined @- prefix | TestResolve_AtDashPrefix | Opsfile line: `@-echo hello` | `ResolvedLine{Text: "echo hello", IgnoreError: true, Silent: true}` | automated (unit) |
| - with variable sub | TestResolve_DashPrefixWithVariable | `-echo $(VAR)` with `VAR=hello` | `ResolvedLine{Text: "echo hello", IgnoreError: true}` | automated (unit) |
| - with env-scoped var | TestResolve_DashPrefixWithScopedVariable | `-echo $(ACCT)` with `prod_ACCT=123` | `ResolvedLine{Text: "echo 123", IgnoreError: true}` | automated (unit) |
| - with backslash cont. | TestResolve_DashPrefixWithBackslashContinuation | `-aws logs \` / `--follow` | Single `ResolvedLine{Text: "aws logs --follow", IgnoreError: true}` | automated (unit) |
| - with indent cont. | TestResolve_DashPrefixWithIndentContinuation | `-aws logs` / (indented) `--follow` | Single `ResolvedLine{Text: "aws logs --follow", IgnoreError: true}` | automated (unit) |
| Executor ignores error | TestExecute_IgnoreErrorContinues | `ResolvedLine{Text: "false", IgnoreError: true}` then `ResolvedLine{Text: "true"}` | No error returned, both lines execute | automated (unit) |
| Executor ignores non-zero exit | TestExecute_IgnoreErrorExitCode42 | `ResolvedLine{Text: "exit 42", IgnoreError: true}` | No error returned | automated (unit) |
| Mixed - and non- lines | TestResolve_MixedDashAndNonDash | Lines: `-docker stop app`, `docker run app` | First IgnoreError=true, second IgnoreError=false | automated (unit) |
| - with @ mixed lines | TestResolve_MultiLineMixedDashAt | Lines: `-@echo setup`, `echo deploy`, `-echo cleanup` | Three lines with correct Silent/IgnoreError flags | automated (unit) |
| IgnoreError + Silent combined | TestExecute_IgnoreErrorAndSilent | `ResolvedLine{Text: "false", IgnoreError: true, Silent: true}` | No echo, no error, execution continues | automated (unit) |
| Dry-run with - | DryRunDashPrefix | `-docker stop app` with `--dry-run` | Prints `docker stop app` (- stripped) | manual verification |

---

## 5. Red Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| Non-dash line still fails | TestExecute_NonDashLineStillFails | `ResolvedLine{Text: "false", IgnoreError: false}` | Error returned with exit code 1 | automated (unit) |
| IgnoreError + invalid shell | TestExecute_IgnoreErrorInvalidShell | `ResolvedLine{Text: "echo hi", IgnoreError: true}`, invalid shell path | Error returned (shell binary not found cannot be ignored -- it's not a command exit code) | automated (unit) |
| - with missing variable | TestResolve_DashPrefixMissingVariable | `-echo $(MISSING)` | Error: variable not defined (resolver error, not execution) | automated (unit) |
| Fail-fast after ignored error | TestExecute_FailAfterIgnored | `{Text: "false", IgnoreError: true}`, then `{Text: "false", IgnoreError: false}` | First error ignored, second causes failure with exit code 1 | automated (unit) |

---

## 6. Edge Cases & Boundary Conditions

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| Double -- prefix | TestResolve_DoubleDashPrefix | `--echo hello` | `ResolvedLine{Text: "-echo hello", IgnoreError: true}` -- one `-` stripped, one remains as shell text | automated (unit) |
| - only (no command) | TestResolve_DashPrefixOnly | Line is just `-` | `ResolvedLine{Text: "", IgnoreError: true, Silent: false}` -- empty command | automated (unit) |
| -@ only (no command) | TestResolve_DashAtPrefixOnly | Line is just `-@` | `ResolvedLine{Text: "", IgnoreError: true, Silent: true}` | automated (unit) |
| - in middle of line | TestResolve_DashInMiddleOfLine | `kubectl delete --force` | `ResolvedLine{Text: "kubectl delete --force", IgnoreError: false}` -- only leading - is special | automated (unit) |
| - in variable value | TestResolve_DashInVariableValue | `VAR=hello-world`, line: `echo $(VAR)` | `ResolvedLine{Text: "echo hello-world", IgnoreError: false}` | automated (unit) |
| - on continuation (not first) | TestResolve_DashOnContinuationFragment | `aws logs \` / `-follow` | `-` on non-first fragment is shell text: `ResolvedLine{Text: "aws logs -follow", IgnoreError: false}` | automated (unit) |
| @ still works alone | TestResolve_AtPrefixStillWorks | `@echo hello` | `ResolvedLine{Text: "echo hello", Silent: true, IgnoreError: false}` -- regression check | automated (unit) |
| IgnoreError on command-not-found | TestExecute_IgnoreErrorCommandNotFound | `ResolvedLine{Text: "nonexistent-binary-xyz", IgnoreError: true}` | Error is ignored, execution continues (shell returns 127 for command not found) | automated (unit) |
| IgnoreError with global silent | TestExecute_IgnoreErrorWithGlobalSilent | `ResolvedLine{Text: "false", IgnoreError: true}`, global silent=true | No echo, no error, execution continues | automated (unit) |
| All lines IgnoreError | TestExecute_AllLinesIgnoreError | Multiple failing lines all with IgnoreError=true | No error returned, all lines attempted | automated (unit) |
| IgnoreError echo still shows | TestExecute_IgnoreErrorEchoStillShows | `ResolvedLine{Text: "false", IgnoreError: true, Silent: false}`, global silent=false | Line is echoed before execution, error is ignored | automated (unit) |
| Empty lines list | TestExecute_EmptyLinesUnchanged | Empty `[]ResolvedLine{}` | No error, no output (regression check) | automated (unit) |
| -@ with backslash cont. | TestResolve_DashAtWithBackslashContinuation | `-@aws stop \` / `--force` | Single `ResolvedLine{Text: "aws stop --force", IgnoreError: true, Silent: true}` | automated (unit) |
| - with whitespace after | TestResolve_DashPrefixWhitespaceAfter | Line: `-   ` (- followed by spaces) | `ResolvedLine{Text: "   ", IgnoreError: true}` | automated (unit) |

---

## 9. Existing Automated Tests

- `TestResolve_AtPrefixStripped` in `internal/command_resolver_test.go` (line 499) -- validates `@` stripping; must still pass after refactoring prefix loop
- `TestResolve_MixedAtAndNonAt` in `internal/command_resolver_test.go` (line 525) -- validates mixed `@` and non-`@` lines
- `TestResolve_DoubleAtPrefix` in `internal/command_resolver_test.go` (line 600) -- validates `@@` strips one `@`; must still pass with loop-based stripping
- `TestResolve_AtPrefixWithBackslashContinuation` in `internal/command_resolver_test.go` (line 572) -- validates `@` with backslash continuation
- `TestResolve_AtOnContinuationFragment` in `internal/command_resolver_test.go` (line 711) -- validates `@` on non-first fragment is shell text
- `TestExecute` in `internal/executor_test.go` (line 23) -- 7 subtests for basic execution and error handling; must still pass
- `TestExecute_ErrorWrapsCommandString` in `internal/executor_test.go` (line 83) -- validates error wrapping
- `TestExecute_EchoesNonSilentLine` in `internal/executor_test.go` (line 107) -- validates echo behavior
- `TestExecute_SkipsSilentLine` in `internal/executor_test.go` (line 116) -- validates silent echo suppression
- `TestExecute_GlobalSilentSuppressesAll` in `internal/executor_test.go` (line 125) -- validates `--silent` flag
- `TestExecute_AtPrefixOnFailingCommand` in `internal/executor_test.go` (line 156) -- validates `@` on a failing command still propagates error

---

## 10. Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| `-` prefix stripped and `IgnoreError` flag set in resolver | unit | Core `-` detection logic in refactored `Resolve()` prefix loop |
| Combined `-@` and `@-` both produce `Silent=true, IgnoreError=true` | unit | Order-independent combined prefix handling |
| `--` double prefix strips only one `-` | unit | FR-5: single-layer stripping, consistent with `@@` behavior |
| `-` with variable substitution | unit | `-` stripped before `substituteVars()` is called |
| `-` with backslash continuation | unit | Parser joins fragments, `-` on first fragment marks whole line IgnoreError |
| `-` mid-line (e.g., `--force` flag) not stripped | unit | FR-6: only leading `-` is Opsfile syntax |
| Executor continues after `IgnoreError=true` line fails | unit | Core ignore-error execution behavior |
| Executor still fails on non-`IgnoreError` line failure | unit | Regression: fail-fast behavior unchanged for non-prefixed lines |
| Executor handles IgnoreError + Silent combined | unit | Orthogonal concerns work together without interference |
| `--dry-run` prints `line.Text` with `-` already stripped | e2e | FR-4: dry-run compatibility |
| Existing `@` prefix tests pass with refactored loop | unit | Regression: `@` behavior unchanged after loop refactor |
| Existing executor error tests pass with `IgnoreError` field | unit | Regression: fail-fast behavior unchanged when `IgnoreError=false` |
| IgnoreError on invalid shell binary (not command exit) | unit | Clarify behavior: shell-not-found errors may or may not be ignorable |

---

## Notes
- **IgnoreError on invalid shell path**: The design doc proposes `if !line.IgnoreError { return error }` which would ignore ALL errors including shell-binary-not-found. The implementation should clarify whether `IgnoreError` only ignores `*exec.ExitError` (command returned non-zero) or all errors from `cmd.Run()` (including shell not found, permission denied). Recommendation: only ignore `*exec.ExitError` to match Make behavior, where `-` ignores the command's exit code but not system-level failures. This is a potential design gap worth discussing.
- **Prefix stripping loop correctness**: The proposed loop handles `@-`, `-@`, `@`, `-`, and bare lines. Verify that the loop terminates correctly for inputs like `-@-` (should strip one `-` and one `@`, leaving `-` as shell text) and `@-@` (should strip one `@` and one `-`, leaving `@` as shell text).
- **Regression risk**: The refactoring of the prefix stripping from a single `if strings.HasPrefix(line, "@")` to a loop changes existing behavior for `@` prefix detection. All 20+ existing `@` tests must continue passing without modification to confirm no regression.
- **`toLines` test helper**: The existing `toLines()` helper in `executor_test.go` creates `ResolvedLine` with `Silent=false`. It should be verified that it also defaults `IgnoreError` to `false` (Go zero-value should handle this, but worth a quick check).
