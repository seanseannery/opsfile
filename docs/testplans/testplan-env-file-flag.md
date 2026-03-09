# Test Plan: `--env-file` / `-e` Flag for Secret Injection

**Feature Doc:** [docs/feature-env-file-flag.md](../feature-env-file-flag.md)
**Issue:** [#23](https://github.com/seanseannery/opsfile/issues/23)

---

## 1. Scope

### In Scope
- Flag parsing: `-e` / `--env-file` short and long forms, single and multiple invocations (`OpsFlags.EnvFiles`)
- Env-file parsing: `ParseEnvFiles` — quoting, comments, blank lines, env-scoped keys, empty values, error cases
- Resolver integration: 6-level priority chain with env-file at levels 3 and 6
- File validation: missing file, unreadable file, directory-instead-of-file — all fail before execution
- Multiple-file ordering: later files override earlier files within the env-file layer
- `--help` text: verifies dry-run secret visibility note is present
- Regression: existing 4-level priority chain tests still pass; no new variables bleed into existing behaviour

### Out of Scope
- Secret masking in `--dry-run` output (documented non-goal)
- Shell expansion / substitution inside `.env` values
- `export KEY=value` syntax support
- `KEY` lines without `=` (bash-style extension)
- System-wide or per-user default env file paths

---

## 2. Test Objectives

- Verify all documented flag forms (`-e`, `--env-file`) are parsed correctly and populate `OpsFlags.EnvFiles` in order
- Verify `ParseEnvFiles` correctly handles the full `.env` format: quoting rules, comments, blank lines, empty values, parse errors
- Verify the 6-level priority chain correctly orders env-file vars at levels 3 and 6 relative to all other sources
- Verify file-not-found / unreadable errors surface before any command execution
- Verify the `--help` output contains a note about `--dry-run` exposing secret values
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
- [ ] `--help` output manually verified to include dry-run note
- [ ] No regressions in related features (flag parsing, command resolution, Opsfile parsing)
- [ ] Test coverage does not decrease

### Acceptance Criteria

- [ ] `-e path` and `--env-file path` both populate `OpsFlags.EnvFiles`
- [ ] Multiple `-e` flags are collected in declaration order
- [ ] Missing or unreadable env-file produces a clear error including the file path, before any command runs
- [ ] Env-file vars are resolved at priority 3 (env-scoped) and 6 (unscoped) — below Opsfile and shell, above "not found"
- [ ] Later files override earlier files for the same key (within the env-file layer)
- [ ] `--help` output contains a note that `--dry-run` will print resolved secret values
- [ ] Env-file variable values are never printed to stdout/stderr by `ops` outside of `--dry-run`

---

## 4. Green Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| `-e` short form | `TestParseOpsFlags_EnvFileShortForm` | `[]string{"-e", ".env", "prod", "cmd"}` | `OpsFlags.EnvFiles == [".env"]`, positionals `["prod","cmd"]` | automated (unit) |
| `--env-file` long form | `TestParseOpsFlags_EnvFileLongForm` | `[]string{"--env-file", ".env", "prod", "cmd"}` | `OpsFlags.EnvFiles == [".env"]`, positionals `["prod","cmd"]` | automated (unit) |
| Multiple `-e` flags in order | `TestParseOpsFlags_EnvFileMultiple` | `[]string{"-e", "a.env", "-e", "b.env", "prod", "cmd"}` | `OpsFlags.EnvFiles == ["a.env","b.env"]` in declaration order | automated (unit) |
| `-e` combined with other flags | `TestParseOpsFlags_EnvFileCombined` | `[]string{"-d", "-e", ".env", "prod", "cmd"}` | `DryRun==true`, `EnvFiles==[".env"]` | automated (unit) |
| Single file, basic vars | `TestParseEnvFiles_SingleFile` | File: `KEY=value\nOTHER=thing` | `OpsVariables{"KEY":"value","OTHER":"thing"}`, no error | automated (unit) |
| Double-quoted value | `TestParseEnvFiles_DoubleQuoted` | File: `DB_PASS="my secret"` | `OpsVariables{"DB_PASS":"my secret"}` | automated (unit) |
| Single-quoted value | `TestParseEnvFiles_SingleQuoted` | File: `TOKEN='sk-abc'` | `OpsVariables{"TOKEN":"sk-abc"}` | automated (unit) |
| Comment lines skipped | `TestParseEnvFiles_CommentsSkipped` | File: `# ignored\nKEY=val` | `OpsVariables{"KEY":"val"}` only | automated (unit) |
| Blank lines skipped | `TestParseEnvFiles_BlankLinesSkipped` | File: `KEY=val\n\nOTHER=x` | Both vars parsed, no error | automated (unit) |
| Env-scoped key parsed | `TestParseEnvFiles_EnvScopedKey` | File: `prod_API_KEY=secret` | `OpsVariables{"prod_API_KEY":"secret"}` | automated (unit) |
| Multiple files, no conflict | `TestParseEnvFiles_MultipleFiles_NoConflict` | file1: `A=1`, file2: `B=2` | `OpsVariables{"A":"1","B":"2"}` | automated (unit) |
| Multiple files, later overrides | `TestParseEnvFiles_MultipleFiles_LaterOverrides` | file1: `VAR=first`, file2: `VAR=second` | `OpsVariables{"VAR":"second"}` | automated (unit) |
| Level 3: env-file scoped resolved | `TestResolveVar_EnvFileScopedFallback` | env-file has `prod_SECRET=env-scoped`, no Opsfile or shell var | `$(SECRET)` resolves to `"env-scoped"` | automated (unit) |
| Level 6: env-file unscoped resolved | `TestResolveVar_EnvFileUnscopedFallback` | env-file has `SECRET=from-file`, no Opsfile or shell var | `$(SECRET)` resolves to `"from-file"` | automated (unit) |
| Level 3 beats level 4 | `TestResolveVar_EnvFileScopedBeatsOpsfileUnscoped` | env-file: `prod_VAR=L3`, Opsfile: `VAR=L4` | resolves to `"L3"` | automated (unit) |
| Level 5 beats level 6 | `TestResolveVar_ShellUnscopedBeatsEnvFileUnscoped` | shell env: `VAR=L5`, env-file: `VAR=L6` | resolves to `"L5"` | automated (unit) |
| Level 4 beats level 6 | `TestResolveVar_OpsfileUnscopedBeatsEnvFileUnscoped` | Opsfile: `VAR=L4`, env-file: `VAR=L6` | resolves to `"L4"` | automated (unit) |
| Level 1 beats all env-file levels | `TestResolveVar_OpsfileScopedBeatsEnvFile` | Opsfile: `prod_VAR=L1`, env-file: `prod_VAR=L3`, env-file: `VAR=L6` | resolves to `"L1"` | automated (unit) |
| Vars from different sources in same command | `TestResolveVar_EnvFileMixedSources` | Opsfile: `A=opsfile`, env-file: `B=envfile` | `echo $(A) $(B)` → `"echo opsfile envfile"` | automated (unit) |
| Empty EnvFiles slice is no-op | `TestParseEnvFiles_EmptySlice` | `ParseEnvFiles([]string{})` | returns empty `OpsVariables{}`, no error | automated (unit) |
| `-h` output contains dry-run note | `TestParseOpsFlags_HelpContainsDryRunNote` | `ParseOpsFlags([]string{"-h"}, &buf)` | `buf` contains text about `--dry-run` and secrets / secret values visible | automated (unit) |

---

## 5. Red Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| Missing file | `TestParseEnvFiles_MissingFile` | `ParseEnvFiles([]string{"/no/such/file.env"})` | error, message contains `"/no/such/file.env"` | automated (unit) |
| Unreadable file | `TestParseEnvFiles_UnreadableFile` | Create file with mode `0000`, call `ParseEnvFiles` | error, message contains file path | automated (unit) |
| Empty key (parse error) | `TestParseEnvFiles_EmptyKey` | File: `=value` | error, message contains line number | automated (unit) |
| Missing path argument | `TestParseOpsFlags_EnvFileMissingArg` | `[]string{"-e"}` (no path follows) | error containing `"flag needs an argument"` or equivalent | automated (unit) |
| `--env-file` missing path argument | `TestParseOpsFlags_EnvFileLongMissingArg` | `[]string{"--env-file"}` | error containing `"flag needs an argument"` or equivalent | automated (unit) |
| Var absent from all sources including env-file | `TestResolveVar_AbsentFromAllSourcesWithEnvFile` | env-file present but `MISSING_VAR` not in it | error contains `"not defined"` | automated (unit) |
| File execution blocked on missing env-file | `TestMain_EnvFileMissingBlocksExecution` | `ops -e /missing.env prod cmd` with valid Opsfile | error surfaces before command execution (no shell exec occurs) | e2e / manual verification |

---

## 6. Edge Cases & Boundary Conditions

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| Empty value | `TestParseEnvFiles_EmptyValue` | File: `KEY=` | `OpsVariables{"KEY":""}`, no error | automated (unit) |
| Whitespace-only value (unquoted) | `TestParseEnvFiles_WhitespaceValue` | File: `KEY=   ` | value trimmed to `""` OR preserved as `"   "` — document which and assert consistently | automated (unit) |
| Inline comment stripped | `TestParseEnvFiles_InlineComment` | File: `KEY=value # comment` | if `indexComment` strips it: `OpsVariables{"KEY":"value"}` — verify behaviour matches Opsfile vars | automated (unit) |
| Whitespace-only lines | `TestParseEnvFiles_WhitespaceOnlyLines` | File: `KEY=val\n   \nOTHER=x` | both vars parsed, whitespace-only line treated as blank | automated (unit) |
| Key with leading/trailing spaces | `TestParseEnvFiles_KeyWithSpaces` | File: `  KEY  =value` | define expected: error OR trimmed `KEY`. Document and test. | automated (unit) |
| `export KEY=value` line | `TestParseEnvFiles_ExportSyntax` | File: `export KEY=value` | per non-goals: parse error OR ignore (key `"export KEY"` or error) — document and test | automated (unit) |
| `KEY` without `=` | `TestParseEnvFiles_KeyWithoutEquals` | File: `NODEFINEDVALUE` | per non-goals: error or skip — document and test | automated (unit) |
| Windows line endings (`\r\n`) | `TestParseEnvFiles_WindowsLineEndings` | File with `\r\n` line endings | vars parsed correctly, no `\r` in values | automated (unit) |
| Same file specified twice | `TestParseEnvFiles_SameFileTwice` | `ParseEnvFiles([]string{"a.env","a.env"})` | second parse overrides first (idempotent), no error | automated (unit) |
| Path is a directory | `TestParseEnvFiles_PathIsDirectory` | `ParseEnvFiles([]string{"/tmp"})` | clear error, includes path | automated (unit) |
| Env-file env-scoped beats Opsfile unscoped (level 3 vs 4) | `TestResolveVar_Level3VsLevel4` | env-file: `prod_VAR=L3`, Opsfile: `VAR=L4`, no shell vars | resolves to `"L3"` — verifies env-file env-scoped is higher priority than Opsfile unscoped | automated (unit) |
| Full 6-level priority chain | `TestResolveVar_FullPriorityChain` | All 6 levels populated; drop each one in turn | confirm each level wins at the correct position in the chain | automated (unit) |
| Env-file env-scoped key for wrong env | `TestResolveVar_EnvFileScopedKeyWrongEnv` | env-file: `staging_VAR=wrong`, running env: `prod` | `$(VAR)` does NOT resolve to `"wrong"` from the staging-scoped key | automated (unit) |
| Empty file | `TestParseEnvFiles_EmptyFile` | File exists but contains no content | `OpsVariables{}`, no error | automated (unit) |
| File with only comments | `TestParseEnvFiles_OnlyComments` | File: `# just a comment\n# another` | `OpsVariables{}`, no error | automated (unit) |
| Large number of variables | `TestParseEnvFiles_ManyVars` | File with 1000 `KEY_N=value_N` lines | all vars parsed, no error | automated (unit) |
| `-e` interleaved with positionals blocked | `TestParseOpsFlags_EnvFlagAfterPositional` | `[]string{"prod", "-e", ".env", "cmd"}` | `-e` treated as positional (not a flag) due to `SetInterspersed(false)` | automated (unit) |

---

## 9. Existing Automated Tests

- `TestParseOpsFlags` in `internal/flag_parser_test.go` (line 12) — 27 subtests covering all existing flags; must be extended for `-e`/`--env-file` cases
- `TestParseOpsFlags_HelpOutput` in `internal/flag_parser_test.go` (line 217) — verifies `-D,-d,-l,-s,-v` in help; must be extended to check `-e` and dry-run note
- `TestResolveVar_PriorityChain` in `internal/command_resolver_test.go` (line 400) — covers 4-level chain (L1–L4); must be extended or supplemented for the new 6-level chain; **note: existing L3/L4 labels mapped to Opsfile unscoped / shell unscoped — these stay valid but new tests must add L3=env-file-scoped and L6=env-file-unscoped**
- `TestResolveVar_UnscopedShellEnvFallback` through `TestResolveVar_ShellEnvUnscopedIsLastResort` (lines 325–398) — individual priority boundary tests; must remain green after the `Resolve` signature change
- `TestResolveVar_MissingVariableReturnsError` / `TestResolve_VariableNotDefined` (lines 193, 282) — these should still return "not defined" errors when env-file has no matching var either

---

## 10. Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| `ParseEnvFiles` for empty/nil slice | unit | Confirms no-op short-circuit path in `main.go` open question |
| Priority level 3 env-file-scoped beats level 4 Opsfile-unscoped | unit | Critical boundary; not explicitly called out in design task breakdown |
| Full 6-level drop-each-level table test | unit | Systematic proof the chain is correctly ordered |
| Windows `\r\n` handling in env-file parser | unit | Portability edge case not mentioned in design |
| `export KEY=value` and `KEY` (no `=`) handling | unit | Non-goals require defined error/skip behaviour — not specified in detail |
| Inline comment stripping consistency with Opsfile parser | unit | Design says reuse `indexComment` but test needed to confirm identical behaviour |
| Env-file missing at execution (blocked before shell exec) | e2e | Validates the "validate before execute" guarantee from FR-1/NFR-3 |
| `--help` output contains dry-run secret visibility note | unit | FR-4 requires this but `TestParseOpsFlags_HelpOutput` does not check for it |
| Path is a directory passed to `-e` | unit | Undefined in design; should fail cleanly |

---

## Notes

### Design Doc Issues Found

1. **Copy-paste error in example usage (line 69):** `ops -e .env prod -e .env.local prod tail-logs` contains `"prod"` twice as a positional argument. Should be: `ops -e .env -e .env.local prod tail-logs`.

2. **FR-4 internal inconsistency:** "Values injected from env-file are **never printed** to stdout or stderr by `ops` itself" but the very next sentence says `--dry-run` resolves and prints them. The intent is clear but the wording is contradictory. Recommended rewrite: "Values from env-file are not printed by `ops` outside of `--dry-run`; `--dry-run` resolves all variable references (including secrets) and prints the resulting shell lines."

3. **Error message format not specified:** NFR-3 says the error message must include the file path but gives no example format. Consider defining a format string in the design doc (e.g., `env-file: open %s: no such file`) to make test assertions less brittle across implementations.

4. **Open question not resolved in FR section:** The open question about `ParseEnvFiles([]string{})` vs skipping entirely is mentioned inline under Key Design Decisions but not captured as a formal spec. This should be resolved before implementation starts.

5. **Existing `TestResolveVar_PriorityChain` level labels:** The test uses "L1"–"L4" labels that will shift meaning when the new levels 3 and 6 are inserted. The test logic will still be correct (the test doesn't use the labels internally) but comments/names should be updated in the PR to avoid confusion.

6. **`-e` flag after positional:** Because `SetInterspersed(false)` stops flag parsing at the first non-flag arg, `ops prod -e .env cmd` will treat `-e .env` as positionals and pass them to `ParseOpsArgs`, causing a confusing error. This is a UX footgun worth documenting in `--help` or the README, even if not changed.
