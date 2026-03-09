# Test Plan: @ Prefix to Suppress Command Echoing

**Feature Doc:** [docs/feature-at-prefix-suppress.md](../feature-at-prefix-suppress.md)

---

## 1. Scope

### In Scope
- `@` prefix detection and stripping in the command resolver
- `ResolvedLine` type with `Silent` metadata
- Default command echoing to stderr in the executor
- `--silent` flag global suppression of echoing
- `--dry-run` compatibility (prints resolved lines with `@` already stripped)
- Backslash continuation with `@` prefix
- Indent continuation with `@` prefix
- Variable substitution on lines with `@` prefix
- Regression testing for existing resolver and executor behavior

### Out of Scope
- Suppression of command stdout/stderr output
- Colorized or formatted echo output
- Per-command block-level suppression

---

## 2. Test Objectives

- Verify that `@` prefix is correctly stripped from resolved command lines and the `Silent` flag is set
- Verify that the executor echoes non-silent lines to stderr and skips echoing for silent lines
- Verify `--silent` flag suppresses all echoing regardless of `@` prefix
- Verify `--dry-run` prints all resolved lines (with `@` stripped) and is unaffected by `@`
- Verify no regressions in existing resolver, parser, or executor behavior after the `ResolvedLine` type change

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

- [ ] Lines without `@` are echoed to stderr before execution
- [ ] Lines with `@` prefix are NOT echoed, but still execute normally
- [ ] `@` is stripped before the line is passed to the shell — it is not shell syntax
- [ ] `--silent` suppresses all echoing regardless of `@` presence
- [ ] `--dry-run` prints all resolved lines with `@` already stripped
- [ ] `--dry-run --silent` prints nothing (existing behavior preserved)
- [ ] `@@cmd` strips one `@` and executes `@cmd` (single-layer stripping)
- [ ] Variable substitution works correctly on `@`-prefixed lines
- [ ] Backslash continuation lines with `@` on the first fragment produce a single silent resolved line
- [ ] No breaking changes to existing Opsfile parsing or command resolution

---

## 4. Green Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| Basic @ strip | TestResolve_AtPrefixStripped | Opsfile line: `@echo hello` | `ResolvedLine{Text: "echo hello", Silent: true}` | automated (unit) |
| No @ prefix | TestResolve_NoAtPrefix | Opsfile line: `echo hello` | `ResolvedLine{Text: "echo hello", Silent: false}` | automated (unit) |
| Mixed @ and non-@ | TestResolve_MixedAtAndNonAt | Lines: `@echo setup`, `aws deploy` | First line Silent=true, second Silent=false | automated (unit) |
| @ with variable sub | TestResolve_AtPrefixWithVariable | `@echo $(VAR)` with `VAR=hello` | `ResolvedLine{Text: "echo hello", Silent: true}` | automated (unit) |
| @ with env-scoped var | TestResolve_AtPrefixWithScopedVariable | `@echo $(ACCT)` with `prod_ACCT=123` | `ResolvedLine{Text: "echo 123", Silent: true}` | automated (unit) |
| @ with backslash cont. | TestResolve_AtPrefixWithBackslashContinuation | `@aws logs \` / `--follow` | Single `ResolvedLine{Text: "aws logs --follow", Silent: true}` | automated (unit) |
| Executor echoes non-silent | TestExecute_EchoesNonSilentLine | `ResolvedLine{Text: "true", Silent: false}`, silent=false | Line text printed to stderr before execution | automated (unit) |
| Executor skips silent line | TestExecute_SkipsSilentLine | `ResolvedLine{Text: "true", Silent: true}`, silent=false | No echo to stderr, command still executes | automated (unit) |
| Executor global silent | TestExecute_GlobalSilentSuppressesAll | Non-silent lines, silent=true | No echo to stderr for any line | automated (unit) |
| Dry-run with @ | DryRunAtPrefix | `@echo hello` with `--dry-run` | Prints `echo hello` to stdout (@ stripped) | manual verification |
| Dry-run+silent with @ | DryRunSilentAtPrefix | `@echo hello` with `--dry-run --silent` | Prints nothing | manual verification |
| Default echo to stderr | DefaultEchoStderr | `echo hello` (no @), default execution | Command echoed to stderr, output to stdout | manual verification |

---

## 5. Red Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| @ on failing command | TestExecute_AtPrefixOnFailingCommand | `ResolvedLine{Text: "false", Silent: true}` | Command fails with exit code 1, no echo before failure | automated (unit) |
| @ with missing variable | TestResolve_AtPrefixMissingVariable | `@echo $(MISSING)` | Error: variable not defined (same as without @) | automated (unit) |
| @ with invalid shell | TestExecute_AtPrefixInvalidShell | Silent line, invalid shell path | Error from exec, no echo attempted | automated (unit) |

---

## 6. Edge Cases & Boundary Conditions

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| Double @@ prefix | TestResolve_DoubleAtPrefix | `@@echo hello` | `ResolvedLine{Text: "@echo hello", Silent: true}` — one @ stripped, one remains as shell text | automated (unit) |
| @ only (no command) | TestResolve_AtPrefixOnly | Line is just `@` | `ResolvedLine{Text: "", Silent: true}` — empty command | automated (unit) |
| @ with leading spaces | TestResolve_AtPrefixAfterSpaces | Raw line `  @echo hi` (spaces before @) | Depends on parser behavior: if TrimSpace is applied first, @ is detected; otherwise treated as shell text. Validate which. | automated (unit) |
| @ with indent continuation | TestResolve_AtPrefixWithIndentContinuation | `@aws logs` / (indented) `--follow` | Single `ResolvedLine{Text: "aws logs --follow", Silent: true}` | automated (unit) |
| @ on continuation line (not first) | TestResolve_AtOnContinuationFragment | `aws logs \` / `@--follow` | The `@` is on a continuation fragment, not the first line. Validate: should the `@` be treated as part of shell text? | automated (unit) |
| Non-identifier $() with @ | TestResolve_AtPrefixNonIdentifierPassthrough | `@$(shell cmd)` | `ResolvedLine{Text: "$(shell cmd)", Silent: true}` — passthrough preserved | automated (unit) |
| @ in middle of line | TestResolve_AtInMiddleOfLine | `echo user@host.com` | `ResolvedLine{Text: "echo user@host.com", Silent: false}` — only leading @ is special | automated (unit) |
| @ in variable value | TestResolve_AtInVariableValue | `VAR=user@host`, line: `echo $(VAR)` | `ResolvedLine{Text: "echo user@host", Silent: false}` — @ in var value is not stripped | automated (unit) |
| Empty lines list + silent | TestExecute_EmptyLinesWithSilent | Empty `[]ResolvedLine`, silent=true | No error, no output | automated (unit) |
| Multi-line: some @ some not | TestResolve_MultiLineMixedAt | Lines: `@echo setup`, `echo deploy`, `@echo cleanup` | Three ResolvedLines with Silent: true, false, true respectively | automated (unit) |
| @ with whitespace-only after | TestResolve_AtPrefixWhitespaceOnlyAfter | Line: `@   ` (@ followed by spaces) | `ResolvedLine{Text: "   ", Silent: true}` or `{Text: "", Silent: true}` depending on trimming | automated (unit) |
| @ with backslash at EOF | TestResolve_AtPrefixBackslashTrailingEOF | `@aws logs \` at EOF (no next line) | Single `ResolvedLine{Text: "aws logs ", Silent: true}` — matches existing backslash-at-EOF behavior | automated (unit) |

---

## 9. Existing Automated Tests

- `TestResolve_ExactEnvMatch` in `internal/command_resolver_test.go` (line 19) — references `got.Lines` as `[]string`; must be updated for `[]ResolvedLine`
- `TestResolve_MultiLineCommand` in `internal/command_resolver_test.go` (line 295) — 2-line assertion on `got.Lines`; must be updated
- All ~30 tests in `internal/command_resolver_test.go` that access `got.Lines[0]` or `got.Lines` — must migrate to use `.Text` field or a `lineTexts()` helper
- `TestExecute` in `internal/executor_test.go` (line 12) — 7 subtests calling `Execute([]string{...}, shell)`; must update to `Execute([]ResolvedLine{...}, shell, silent)`
- `TestExecute_ErrorWrapsCommandString` in `internal/executor_test.go` (line 72) — single call; must update signature
- `TestExecute_InvalidShellPath` in `internal/executor_test.go` (line 78) — must update signature
- `TestExecute_CommandWithPipe` in `internal/executor_test.go` (line 83) — must update signature
- `TestExecute_StderrConnected` in `internal/executor_test.go` (line 88) — must update signature; note: this test validates stderr output from commands, which may now include echoed command text mixed in

---

## 10. Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| `@` prefix stripped and `Silent` flag set in resolver | unit | Core `@` detection logic in `Resolve()` |
| Mixed `@` and non-`@` lines produce correct `Silent` flags | unit | Per-line independence of `@` detection |
| `@@` double prefix strips only one `@` | unit | NFR-3: single-layer stripping |
| `@` with variable substitution | unit | `@` stripped before `substituteVars()` |
| `@` with backslash continuation | unit | Parser joins fragments, `@` on first fragment marks whole line silent |
| `@` mid-line (e.g., email address) is not stripped | unit | Only leading `@` is Opsfile syntax |
| Executor echoes to stderr when `!silent && !line.Silent` | unit | Default echoing behavior (capture stderr in test) |
| Executor skips echo when `line.Silent == true` | unit | Per-line suppression |
| Executor skips all echo when `silent == true` | unit | Global `--silent` override |
| `--dry-run` prints `line.Text` with `@` already stripped | e2e | Dry-run compatibility |
| `--dry-run --silent` prints nothing | e2e | Combined flag behavior |
| Existing executor tests pass with updated signature | unit | Regression — no behavior change for existing commands |
| Existing resolver tests pass with `ResolvedLine` type | unit | Regression — no behavior change for existing resolution |

---

## Notes
- **Breaking behavior change**: This feature introduces default command echoing to stderr. All existing Opsfile users will see this new output. The `--silent` flag restores the previous quiet behavior. This should be prominently documented in release notes.
- **Test migration**: The `ResolvedCommand.Lines` type change from `[]string` to `[]ResolvedLine` requires updating every existing test that accesses `.Lines`. A `lineTexts(rc ResolvedCommand) []string` helper function in the test file would minimize churn and keep assertions readable.
- **Stderr capture in executor tests**: Testing echo-to-stderr requires capturing stderr output in tests. Consider using an `io.Writer` parameter or a package-level `var echoWriter io.Writer = os.Stderr` that tests can override, rather than hardcoding `os.Stderr` in the `Execute()` function. The current design doc specifies `fmt.Fprintln(os.Stderr, ...)` which is not testable without OS-level pipe redirection.
- **`TestExecute_StderrConnected` interaction**: This existing test validates that command stderr output works. After this change, the executor will also write echo output to stderr. The test should still pass (it only checks for no error), but be aware that stderr now has mixed content (echo + command stderr).
- **Open question from design doc**: Whether echoed lines should have a visual prefix (e.g., `+ command` or `$ command`). Test plan assumes no prefix per the current proposal. If a prefix is added, echo-related tests would need to account for it.
