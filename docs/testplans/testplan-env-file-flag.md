# Test Plan: `--env-file` / `-e` Flag for Secret Injection

**Feature Doc:** [docs/feature-env-file-flag.md](../feature-env-file-flag.md)
**Issue:** [#23](https://github.com/seanseannery/opsfile/issues/23)

---

## 1. Scope

### In Scope
- Flag parsing: `-e` / `--env-file` short and long forms; `OpsFlags.EnvFile string` (single value); error if specified more than once
- Default env file: auto-loading `.ops_secrets.env` from the Opsfile directory when `-e` is not given; silent skip when absent; bypass when `-e` is explicit
- Env-file parsing: `ParseEnvFile` — quoting, comments, blank lines, env-scoped keys, empty values, error cases
- Resolver integration: updated 6-level priority chain (shell env wins) with env-file at levels 3 and 6
- File validation: missing explicit file errors before execution; missing default file is silently skipped
- `--help` text: dry-run secret visibility note and flag-position constraint (`-e` must precede positionals)
- Regression: existing resolver priority tests still pass after shell-wins reorder; no `EnvFiles []string` references remain

### Out of Scope
- Secret masking in `--dry-run` output (documented non-goal)
- Shell expansion / substitution inside `.env` values
- `export KEY=value` syntax support
- `KEY` lines without `=` (bash-style extension)
- Stacking `.ops_secrets.env` with an explicit `-e` file (non-goal: explicit replaces default)

---

## 2. Test Objectives

- Verify `-e` / `--env-file` is parsed correctly as a single-value `string` field, and errors on duplicate use
- Verify `.ops_secrets.env` is auto-loaded when absent and silently skipped when absent, and skipped entirely when `-e` is explicit
- Verify `ParseEnvFile` correctly handles the full `.env` format: quoting rules, comments, blank lines, empty values, parse errors
- Verify the updated 6-level priority chain places shell env above Opsfile and env-file at the bottom two slots
- Verify explicit env-file missing errors include the path in format `env-file "<path>": <os error>`
- Verify `--help` output contains both the dry-run secret note and the flag-position constraint note
- Verify no regressions in flag parsing, command resolution, or Opsfile variable handling

---

## 3. Entry & Exit Criteria

### Entry Criteria (prerequisites before testing begins)
- [ ] Feature implementation is code-complete
- [ ] Code compiles without errors (`make build`)
- [ ] Existing tests still pass (`make test`)

### Exit Criteria (conditions to consider testing complete)
- [ ] All green path automated tests pass
- [ ] All red path automated tests pass
- [ ] All edge case automated tests pass
- [ ] `--help` output manually verified to include dry-run note and flag-position constraint
- [ ] No regressions in related features (flag parsing, command resolution, Opsfile parsing)
- [ ] Test coverage does not decrease

### Acceptance Criteria

- [ ] `-e path` and `--env-file path` both set `OpsFlags.EnvFile` (a `string`, not a slice)
- [ ] Specifying `-e` more than once is an error
- [ ] `.ops_secrets.env` in the Opsfile directory is auto-loaded when no `-e` flag is given and the file exists
- [ ] `.ops_secrets.env` absence (when no `-e` given) is silently ignored — no error, no warning
- [ ] When `-e` is explicit, `.ops_secrets.env` is not loaded even if present
- [ ] Missing explicit env-file produces error in format `env-file "<path>": <os error>` before any command runs
- [ ] Shell env-scoped wins at priority 1, Opsfile env-scoped at 2, env-file env-scoped at 3, shell unscoped at 4, Opsfile unscoped at 5, env-file unscoped at 6
- [ ] `--help` output contains note that `--dry-run` prints resolved secret values
- [ ] `--help` output contains note that `-e` must appear before environment/command positionals

---

## 4. Green Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| `-e` short form | `TestParseOpsFlags_EnvFileShortForm` | `[]string{"-e", ".env", "prod", "cmd"}` | `OpsFlags.EnvFile == ".env"`, positionals `["prod","cmd"]` | automated (unit) |
| `--env-file` long form | `TestParseOpsFlags_EnvFileLongForm` | `[]string{"--env-file", ".env", "prod", "cmd"}` | `OpsFlags.EnvFile == ".env"`, positionals `["prod","cmd"]` | automated (unit) |
| `-e` combined with `-d` | `TestParseOpsFlags_EnvFileCombinedWithDryRun` | `[]string{"-d", "-e", ".env", "prod", "cmd"}` | `DryRun==true`, `EnvFile==".env"` | automated (unit) |
| `-e` combined with `-D` | `TestParseOpsFlags_EnvFileCombinedWithDirectory` | `[]string{"-D", "/tmp", "-e", ".env", "prod", "cmd"}` | `Directory=="/tmp"`, `EnvFile==".env"` | automated (unit) |
| No flag, `EnvFile` is empty | `TestParseOpsFlags_NoEnvFile` | `[]string{"prod", "cmd"}` | `OpsFlags.EnvFile == ""` | automated (unit) |
| Single file, basic vars | `TestParseEnvFile_BasicVars` | File: `KEY=value\nOTHER=thing` | `OpsVariables{"KEY":"value","OTHER":"thing"}`, no error | automated (unit) |
| Double-quoted value | `TestParseEnvFile_DoubleQuoted` | File: `DB_PASS="my secret"` | `OpsVariables{"DB_PASS":"my secret"}` | automated (unit) |
| Single-quoted value | `TestParseEnvFile_SingleQuoted` | File: `TOKEN='sk-abc'` | `OpsVariables{"TOKEN":"sk-abc"}` | automated (unit) |
| Comment lines skipped | `TestParseEnvFile_CommentsSkipped` | File: `# ignored\nKEY=val` | `OpsVariables{"KEY":"val"}` only | automated (unit) |
| Blank lines skipped | `TestParseEnvFile_BlankLinesSkipped` | File: `KEY=val\n\nOTHER=x` | both vars parsed, no error | automated (unit) |
| Env-scoped key parsed | `TestParseEnvFile_EnvScopedKey` | File: `prod_API_KEY=secret` | `OpsVariables{"prod_API_KEY":"secret"}` | automated (unit) |
| Default `.ops_secrets.env` loaded | `TestMain_DefaultEnvFileLoaded` | `.ops_secrets.env` present in Opsfile dir, no `-e` flag | vars from `.ops_secrets.env` resolve in commands | e2e / manual verification |
| Default `.ops_secrets.env` absent silently | `TestMain_DefaultEnvFileAbsentSilent` | No `.ops_secrets.env` in Opsfile dir, no `-e` flag | no error; run proceeds as if no env-file | e2e / manual verification |
| Explicit `-e` bypasses default | `TestMain_ExplicitEnvFileBypassesDefault` | `.ops_secrets.env` present in Opsfile dir AND `-e other.env` given | only `other.env` vars loaded; `.ops_secrets.env` vars absent | e2e / manual verification |
| Priority P1: shell env-scoped wins all | `TestResolveVar_ShellScopedWinsAll` | shell: `prod_VAR=P1`, Opsfile: `prod_VAR=P2`, env-file: `prod_VAR=P3` | resolves to `"P1"` | automated (unit) |
| Priority P2: Opsfile env-scoped beats env-file scoped and unscoped | `TestResolveVar_OpsfileScopedBeatsEnvFile` | Opsfile: `prod_VAR=P2`, env-file: `prod_VAR=P3`, env-file: `VAR=P6` | resolves to `"P2"` | automated (unit) |
| Priority P3: env-file env-scoped resolved | `TestResolveVar_EnvFileScopedFallback` | env-file: `prod_SECRET=P3`, no Opsfile or shell var | `$(SECRET)` resolves to `"P3"` | automated (unit) |
| Priority P4: shell unscoped beats Opsfile and env-file unscoped | `TestResolveVar_ShellUnscopedBeatsRemainder` | shell: `VAR=P4`, Opsfile: `VAR=P5`, env-file: `VAR=P6` | resolves to `"P4"` | automated (unit) |
| Priority P5: Opsfile unscoped beats env-file unscoped | `TestResolveVar_OpsfileUnscopedBeatsEnvFileUnscoped` | Opsfile: `VAR=P5`, env-file: `VAR=P6` | resolves to `"P5"` | automated (unit) |
| Priority P6: env-file unscoped resolved | `TestResolveVar_EnvFileUnscopedFallback` | env-file: `SECRET=P6`, no Opsfile or shell var | `$(SECRET)` resolves to `"P6"` | automated (unit) |
| Vars from Opsfile and env-file in same command | `TestResolveVar_EnvFileMixedSources` | Opsfile: `A=opsfile`, env-file: `B=envfile` | `echo $(A) $(B)` → `"echo opsfile envfile"` | automated (unit) |
| Shell overrides env-file for same key | `TestResolveVar_ShellOverridesEnvFile` | shell: `VAR=shell`, env-file: `VAR=envfile` | resolves to `"shell"` | automated (unit) |
| Empty `EnvFile` is no-op | `TestParseEnvFile_EmptyFile` | File exists but contains no content | `OpsVariables{}`, no error | automated (unit) |
| `-h` output contains dry-run note | `TestParseOpsFlags_HelpContainsDryRunNote` | `ParseOpsFlags([]string{"-h"}, &buf)` | `buf` contains text about `--dry-run` and secret/secret values visible | automated (unit) |
| `-h` output contains flag-position note | `TestParseOpsFlags_HelpContainsFlagPositionNote` | `ParseOpsFlags([]string{"-h"}, &buf)` | `buf` contains text that `-e` must precede environment/command positionals | automated (unit) |
| `-h` output contains `-e` flag | `TestParseOpsFlags_HelpContainsEnvFlag` | `ParseOpsFlags([]string{"-h"}, &buf)` | `buf` contains `-e` | automated (unit) |

---

## 5. Red Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| `-e` specified twice is error | `TestParseOpsFlags_EnvFileDuplicate` | `[]string{"-e", "a.env", "-e", "b.env", "prod", "cmd"}` | error returned (not ErrHelp); error message indicates duplicate/repeated flag | automated (unit) |
| `--env-file` twice is error | `TestParseOpsFlags_EnvFileLongDuplicate` | `[]string{"--env-file", "a.env", "--env-file", "b.env", "prod", "cmd"}` | error returned | automated (unit) |
| Missing explicit env-file | `TestParseEnvFile_MissingFile` | `ParseEnvFile("/no/such/file.env")` | error matching `env-file "/no/such/file.env": ...`, contains path | automated (unit) |
| Unreadable explicit env-file | `TestParseEnvFile_UnreadableFile` | Create file with mode `0000`, call `ParseEnvFile` | error containing file path | automated (unit) |
| Empty key (parse error) | `TestParseEnvFile_EmptyKey` | File: `=value` | error containing line number | automated (unit) |
| `-e` missing argument (short) | `TestParseOpsFlags_EnvFileMissingArg` | `[]string{"-e"}` | error containing `"flag needs an argument"` or equivalent | automated (unit) |
| `--env-file` missing argument | `TestParseOpsFlags_EnvFileLongMissingArg` | `[]string{"--env-file"}` | error containing `"flag needs an argument"` or equivalent | automated (unit) |
| Var absent from all sources including env-file | `TestResolveVar_AbsentWithEnvFile` | env-file present but `MISSING_VAR` not in it | error contains `"not defined"` | automated (unit) |
| Explicit missing file blocks execution | `TestMain_MissingExplicitEnvFileBlocksExec` | `ops -e /missing.env prod cmd` with valid Opsfile | error in `env-file "/missing.env": ...` format before any shell exec | e2e / manual verification |

---

## 6. Edge Cases & Boundary Conditions

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| Empty value | `TestParseEnvFile_EmptyValue` | File: `KEY=` | `OpsVariables{"KEY":""}`, no error | automated (unit) |
| Whitespace-only value (unquoted) | `TestParseEnvFile_WhitespaceValue` | File: `KEY=   ` | define and assert: trimmed to `""` or preserved — document which | automated (unit) |
| Inline comment stripped | `TestParseEnvFile_InlineComment` | File: `KEY=value # comment` | `OpsVariables{"KEY":"value"}` — same behaviour as Opsfile `indexComment` | automated (unit) |
| Whitespace-only lines | `TestParseEnvFile_WhitespaceOnlyLines` | File: `KEY=val\n   \nOTHER=x` | both vars parsed; whitespace-only line treated as blank | automated (unit) |
| `export KEY=value` line | `TestParseEnvFile_ExportSyntax` | File: `export KEY=value` | define expected: parse error OR treats `"export KEY"` as the name — document and test | automated (unit) |
| `KEY` without `=` | `TestParseEnvFile_KeyWithoutEquals` | File: `NODEFINEDVALUE` | define expected: error or skip — document and test | automated (unit) |
| Windows line endings (`\r\n`) | `TestParseEnvFile_WindowsLineEndings` | File with `\r\n` line endings | vars parsed correctly; no `\r` in values | automated (unit) |
| Path is a directory | `TestParseEnvFile_PathIsDirectory` | `ParseEnvFile("/tmp")` | clear error; message contains path | automated (unit) |
| Env-file env-scoped key for wrong env | `TestResolveVar_EnvFileScopedKeyWrongEnv` | env-file: `staging_VAR=wrong`, running env: `prod` | `$(VAR)` does NOT resolve from the staging-scoped key | automated (unit) |
| P3 beats P5: env-file scoped beats Opsfile unscoped | `TestResolveVar_EnvFileScopedBeatsOpsfileUnscoped` | env-file: `prod_VAR=P3`, Opsfile: `VAR=P5`, no shell vars | resolves to `"P3"` — env-file env-scoped is higher priority than Opsfile unscoped | automated (unit) |
| Full 6-level drop-each table | `TestResolveVar_FullPriorityChain` | Drop each priority level one at a time, assert the next level wins | systematic proof of all 6 priority positions in correct order | automated (unit) |
| `-e` after positional is silently ignored | `TestParseOpsFlags_EnvFlagAfterPositional` | `[]string{"prod", "-e", ".env", "cmd"}` | due to `SetInterspersed(false)`: `-e`, `.env`, `cmd` treated as positionals; `EnvFile == ""` | automated (unit) |
| Default `.ops_secrets.env` present with explicit `-e` | `TestMain_DefaultNotStackedWithExplicit` | `.ops_secrets.env` in Opsfile dir with `VAR=from-default`; `-e other.env` (no `VAR`); command uses `$(VAR)` | `VAR` is NOT resolved from `.ops_secrets.env` — explicit replaces default entirely | e2e / manual verification |
| File with only comments | `TestParseEnvFile_OnlyComments` | File: `# just a comment\n# another` | `OpsVariables{}`, no error | automated (unit) |
| Error message format | `TestParseEnvFile_ErrorFormat` | `ParseEnvFile("/no/such/path.env")` | error message matches `env-file "/no/such/path.env":` prefix exactly | automated (unit) |

---

## 9. Existing Automated Tests

- `TestParseOpsFlags` in `internal/flag_parser_test.go` (line 12) — 27 subtests covering existing flags; must be extended for `-e`/`--env-file` single-value cases and duplicate-flag error
- `TestParseOpsFlags_HelpOutput` in `internal/flag_parser_test.go` (line 217) — verifies `-D,-d,-l,-s,-v` in help; must be extended to check `-e`, dry-run note, and flag-position constraint
- `TestResolveVar_PriorityChain` in `internal/command_resolver_test.go` (line 400) — covers the old 4-level chain with "level1–level4" subtest names; **implementation PR must rename these to avoid collision with new 6-level numbering (suggested: "p1 shell-env-scoped" … "p4 opsfile-unscoped")** and extend for the two new env-file levels
- `TestResolveVar_OpsfileEnvScopedBeatsShellEnvScoped` (line 349) — **this test will FAIL after the priority swap (shell now beats Opsfile)**; it must be inverted or replaced as part of the implementation PR
- `TestResolveVar_ShellEnvScopedBeatsOpsfileUnscoped` (line 362) — still valid (shell-scoped beats Opsfile-unscoped) but now for different reasons; verify it still passes
- `TestResolveVar_OpsfileUnscopedBeatsShellEnvUnscoped` (line 375) — **this test will FAIL after the priority swap (shell-unscoped is now P4, Opsfile-unscoped is P5)**; must be inverted or replaced
- `TestResolveVar_ShellEnvUnscopedIsLastResort` (line 388) — **no longer accurate**; shell-unscoped is now P4 (not last resort); rename and rework

---

## 10. Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| Full 6-level drop-each-level table | unit | Systematic proof all 6 priority positions are correctly ordered after the shell-wins change |
| Shell env beats Opsfile env-scoped (new P1 > P2 boundary) | unit | Critical inversion from old design — three existing tests currently assert the opposite and must be replaced |
| `ParseEnvFile("")` (empty-string path) | unit | Ensures empty `EnvFile` field in `OpsFlags` is guarded in `main.go` before calling `ParseEnvFile` |
| Default `.ops_secrets.env` auto-load (file present) | e2e | Validates FR-5 happy path |
| Default `.ops_secrets.env` absent is silent (no error) | e2e | Validates FR-5 silent-skip guarantee |
| Explicit `-e` suppresses default `.ops_secrets.env` | e2e | Validates FR-5 "explicit replaces default" design decision |
| `-e` specified twice produces error | unit | FR-1 single-use constraint |
| `--help` contains flag-position constraint note | unit | FR-6: `SetInterspersed(false)` footgun must be documented |
| Error message format `env-file "<path>": ...` | unit | NFR-3 specifies exact format; should be asserted, not just "contains path" |
| Windows `\r\n` line endings in env-file | unit | Portability; not mentioned in design but common cross-platform issue |
| Path-is-a-directory passed to `-e` | unit | Undefined in design; should fail cleanly with path in error |

---

## Notes

### Remaining Design Doc Inconsistencies

1. **Key Design Decisions section still mentions `pflag.StringArrayP`** (bottom of section 4): The flag was changed to single-use `StringP` but the decision note still says `StringArrayP`. This is a leftover from the old design. Should be corrected before implementation to avoid confusion when the implementer reads the doc.

2. **Architecture section still says `ParseEnvFiles` (plural)**: The component design and data-flow diagram use `ParseEnvFile` (singular) correctly, but one line in the overview says `call internal.ParseEnvFiles(flags.EnvFiles)`. Should be `ParseEnvFile(flags.EnvFile)`.

3. **Three existing priority tests will break on the shell-wins change**: `TestResolveVar_OpsfileEnvScopedBeatsShellEnvScoped`, `TestResolveVar_OpsfileUnscopedBeatsShellEnvUnscoped`, and `TestResolveVar_ShellEnvUnscopedIsLastResort` all assert the old Opsfile-wins order. The implementation PR must update these — they must not be deleted, only corrected. See Section 9 above for details.

4. **`.ops_secrets.env` gitignore guidance**: FR-5 mentions this should be added to `.gitignore` by convention; confirm this is covered in the `--help` text or README update as part of the implementation scope.
