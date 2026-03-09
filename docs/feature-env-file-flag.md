# Feature: `--env-file` / `-e` Flag for Secret Injection

## 1. Problem Statement & High-Level Goals

### Problem
Opsfile variables are committed to the repository, which is intentional for non-sensitive configuration (cluster names, regions, log groups). However, many operational commands require secrets or credentials that must not be committed — AWS session tokens, kubeconfig paths, API keys, database passwords.

Currently the only workaround is to pre-export these as shell environment variables before running `ops`, which requires the operator to remember which variables are needed and set them manually in their shell session. There is no way to point `ops` at a `.env` file containing those values at invocation time.

Related: [Issue #23](https://github.com/seanseannery/opsfile/issues/23)

### Goals
- [x] Add `-e` / `--env-file <path>` flag, repeatable, collecting paths in order
- [x] Parse `.env`-format files using the same quoting/comment rules as Opsfile variable parsing
- [x] Inject env-file variables into the resolution chain at two new priority levels (below Opsfile and shell env, above "not found")
- [x] Validate file existence before any command executes
- [x] Document `--dry-run` visibility of injected secrets in `--help` output

### Non-Goals
- Encrypting or masking secrets from `--dry-run` output (out of scope; documented as a known behaviour)
- Supporting shell expansion or substitution inside `.env` file values
- Supporting `export` keyword or `KEY` lines without `=` (bash-style .env extensions)
- Adding a system-wide or per-user default env file path

---

## 2. Functional Requirements

### FR-1: Flag Parsing
`-e <path>` and `--env-file <path>` are equivalent. The flag may be specified multiple times; all paths are collected in order into `OpsFlags.EnvFiles []string`. An unknown or missing path produces a clear error **before** any command executes.

### FR-2: File Format
Standard `.env` syntax:
```
# Comments are ignored
AWS_SESSION_TOKEN=AQoXnyc...
DB_PASSWORD="my secret password"
prod_API_KEY='sk-...'
```
- `NAME=value` lines, with the same quoting rules as Opsfile variables (unquoted, single-quoted, double-quoted), handled by the existing `extractVariableValue` helper.
- Lines starting with `#` (after trimming) are skipped.
- Blank lines are skipped.
- A `=value` line with no name (empty name) is a parse error with line number.
- Env-scoped names (`prod_VAR`) are supported and follow the same resolution priority as Opsfile variables.
- When multiple `-e` flags are given, files are processed in order; later files override earlier files **within the env-file layer only**.

### FR-3: Resolution Priority Chain (Updated)

| Priority | Source | Key |
|----------|--------|-----|
| 1 (highest) | Opsfile env-scoped | `vars["env_VAR"]` |
| 2 | Shell env-scoped | `os.LookupEnv("env_VAR")` |
| 3 | Env-file env-scoped | `envFileVars["env_VAR"]` |
| 4 | Opsfile unscoped | `vars["VAR"]` |
| 5 | Shell unscoped | `os.LookupEnv("VAR")` |
| 6 (lowest) | Env-file unscoped | `envFileVars["VAR"]` |

Opsfile and shell environment variables always take precedence over env-file values — the flag is purely additive.

### FR-4: Security / UX
- Values are not printed outside of `--dry-run`; `ops` never echoes raw env-file contents to stdout or stderr.
- `--dry-run` resolves all variable references — including those sourced from env-file — and prints the resulting shell lines. Secret values will therefore be visible in `--dry-run` output. The `--help` text must include a note to this effect.

### FR-5: Flag Position Constraint
Because `SetInterspersed(false)` stops flag parsing at the first positional argument, `-e` flags must appear **before** the environment and command positionals. `ops prod -e .env cmd` will silently ignore `-e .env`. The `--help` output must document this constraint to prevent unexpected "variable not defined" errors.

### Example Usage

```bash
ops -e .env.prod prod rollback
ops --env-file ~/.secrets/prod.env prod tail-logs
ops -e .env -e .env.local prod tail-logs         # multiple files, last wins on conflict
```

`.env.prod`:
```
# AWS credentials for prod
AWS_SESSION_TOKEN=AQoXnyc...
prod_DB_PASSWORD="my-secret"
```

`Opsfile`:
```
CLUSTER=my-cluster

rollback:
  prod:
    aws ecs update-service --cluster $(CLUSTER) --force-new-deployment
    echo "token=$(AWS_SESSION_TOKEN)"
```

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Performance | Env-file parsing adds no perceptible latency | Files are small; single-pass scan |
| NFR-2 | Compatibility | Linux, macOS (same platforms as existing binary) | No platform-specific I/O |
| NFR-3 | Reliability | Missing or unreadable file fails before execution | Error format: `env-file "<path>": <os error>` |
| NFR-4 | Security | File contents never echoed to stdout/stderr | Only resolved variable values reach shell lines |
| NFR-5 | Maintainability | No new external dependencies | Reuse `extractVariableValue`/`indexComment` |
| NFR-6 | Test Coverage | Coverage must not decrease | All acceptance criteria have unit tests |

---

## 4. Architecture & Implementation Proposal

### Overview

The implementation touches three existing files and adds one new file. The key principle is to reuse the Opsfile variable-parsing helpers (`extractVariableValue`, `indexComment`) for env-file parsing, and extend the resolver's `resolveVar` function with a sixth-level fallback map.

### Component Design

**`internal/flag_parser.go`** — add `EnvFiles []string` to `OpsFlags` and register `-e`/`--env-file` as a repeatable `StringArray` flag.

**`internal/envfile_parser.go`** (new) — `ParseEnvFiles(paths []string) (OpsVariables, error)` reads each file in order, parses `NAME=value` lines using `extractVariableValue`, and merges results (later paths override earlier for the same key).

**`internal/command_resolver.go`** — extend `Resolve` and `resolveVar` to accept `envFileVars OpsVariables` and consult it at priority levels 3 and 6.

**`cmd/ops/main.go`** — after flag parsing, call `internal.ParseEnvFiles(flags.EnvFiles)` if `len(flags.EnvFiles) > 0` (skip entirely when the flag is not used to avoid allocating an empty map), and pass the result to `internal.Resolve`.

### Data Flow

```
os.Args
  │
  ▼
ParseOpsFlags()          ← adds EnvFiles []string
  │
  ├─ flags.EnvFiles ──► ParseEnvFiles() ──► envFileVars OpsVariables
  │
  ▼
ParseOpsFile()           ← unchanged, produces vars OpsVariables
  │
  ▼
Resolve(cmd, env, commands, vars, envFileVars)
  │
  └─ resolveVar(name, env, vars, envFileVars)
       1. vars[env_NAME]
       2. os.LookupEnv(env_NAME)
       3. envFileVars[env_NAME]   ← NEW
       4. vars[NAME]
       5. os.LookupEnv(NAME)
       6. envFileVars[NAME]       ← NEW
```

#### Sequence Diagram

```
main()
  │ ParseOpsFlags(os.Args[1:])
  │──────────────────────────► flag_parser.go
  │◄── OpsFlags{EnvFiles: [...]}
  │
  │ [if EnvFiles non-empty]
  │ ParseEnvFiles(flags.EnvFiles)
  │──────────────────────────► envfile_parser.go
  │◄── envFileVars, err
  │    [error if file missing/unreadable]
  │
  │ ParseOpsFile(path)
  │──────────────────────────► opsfile_parser.go
  │◄── vars, commands, ...
  │
  │ Resolve(cmd, env, commands, vars, envFileVars)
  │──────────────────────────► command_resolver.go
  │    resolveVar checks 6 levels
  │◄── ResolvedCommand
  │
  │ Execute(resolved.Lines, ...)
  │──────────────────────────► executor.go
```

### Key Design Decisions

- **Separate `envFileVars` parameter over merged map**: Passing `envFileVars` as a distinct parameter to `resolveVar` makes the priority boundary explicit in code and in tests. Merging into a single map before resolution would require key-name tricks to preserve ordering.
- **Reuse `extractVariableValue`**: The quoting and comment-stripping logic in `opsfile_parser.go` is already correct and tested. `envfile_parser.go` calls it directly rather than duplicating.
- **Validate files before execution**: Checking file existence and readability in `main.go` immediately after `ParseOpsFlags` (before `ParseOpsFile`) ensures operators see a clear error and no command runs with incomplete variable context.
- **`pflag.StringArrayP` for repeatable flag**: `pflag` already supports this; `-e a -e b` produces `[]string{"a","b"}` in declaration order.

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/flag_parser.go` | Modify | Add `EnvFiles []string` to `OpsFlags`; register `-e`/`--env-file` flag; update help text to note `--dry-run` visibility |
| `internal/envfile_parser.go` | Create | `ParseEnvFiles(paths []string) (OpsVariables, error)` — single-pass line scanner, reuses `extractVariableValue` |
| `internal/envfile_parser_test.go` | Create | Unit tests: single file, multiple files, quoting, comments, env-scoped keys, error cases |
| `internal/command_resolver.go` | Modify | Add `envFileVars OpsVariables` param to `Resolve`, `substituteVars`, `resolveVar`; add priority 3 and 6 lookups |
| `internal/command_resolver_test.go` | Modify | Add tests for env-file priority levels 3 and 6; rename existing `TestResolveVar_PriorityChain` subtests from "level1–level4" to "p1–p4" to avoid collision with the new 6-level numbering |
| `internal/flag_parser_test.go` | Modify | Add tests: single and multiple `-e` flags populate `EnvFiles` in order |
| `cmd/ops/main.go` | Modify | Call `ParseEnvFiles` after flag parsing; pass `envFileVars` to `Resolve` |

---

## 5. Alternatives Considered

### Alternative A: Pre-merge env-file vars into `OpsVariables` before calling Resolve

**Description:** Load env-file vars, then merge them into the `vars` map returned by `ParseOpsFile`, using suffixed keys to encode priority (e.g., `__envfile__VAR`).

**Pros:**
- No signature change to `Resolve` or `resolveVar`

**Cons:**
- Encoding priority in key names is brittle and opaque
- Complicates `resolveVar` lookup logic with string prefix checks
- Harder to test priority boundaries in isolation

**Why not chosen:** The explicit separate-parameter approach is cleaner, more readable, and easier to test.

---

### Alternative B: Shell-export the env-file values into the process before running

**Description:** Before calling `Execute`, iterate env-file vars and call `os.Setenv` so they become shell environment variables and fall into existing levels 2 and 4.

**Pros:**
- Zero changes to `resolveVar`

**Cons:**
- Pollutes the current process environment for the lifetime of `ops`
- Cannot implement the correct priority (env-file should be lower than shell env)
- Makes cleanup/isolation impossible

**Why not chosen:** Priority semantics are incorrect and side effects are unacceptable.

---

## Open Questions
None — all questions resolved.

---

## 6. Task Breakdown

### Phase 1: Foundation
- [ ] Add `EnvFiles []string` to `OpsFlags` in `flag_parser.go`
- [ ] Register `-e`/`--env-file` as a repeatable `StringArrayP` flag; update `--help` text
- [ ] Write `ParseEnvFiles` in `internal/envfile_parser.go`
- [ ] Write unit tests in `internal/envfile_parser_test.go`

### Phase 2: Integration
- [ ] Extend `resolveVar` (and callers) to accept and consult `envFileVars`
- [ ] Add resolver tests for priority levels 3 and 6
- [ ] Wire `ParseEnvFiles` into `cmd/ops/main.go`; validate files before execution
- [ ] Add flag-parser tests for `-e` / `--env-file`

### Phase 3: Polish
- [ ] Confirm `--help` output notes `--dry-run` secret visibility and flag-position constraint
- [ ] Update `AGENTS.md` directory structure if new files added
