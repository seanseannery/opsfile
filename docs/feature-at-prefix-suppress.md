# Feature: @ Prefix to Suppress Command Echoing


## 1. Problem Statement & High-Level Goals

### Problem
When `ops` executes commands from an Opsfile, users have no visibility into which shell lines are being run — the tool silently executes each line and only command output (stdout/stderr) is visible. This makes debugging difficult and differs from tools like `make`, which echo commands before execution. Conversely, for polished scripts with user-facing output, echoing setup commands creates noise. Users need both default visibility and granular suppression. (Issue #6)

### Goals
- [ ] Echo each command line to stderr before execution by default
- [ ] Support `@` prefix on Opsfile command lines to suppress echoing for that line
- [ ] Respect `--silent` flag as a global echo suppression override
- [ ] Preserve existing `--dry-run` behavior (prints all resolved lines regardless of `@`)

### Non-Goals
- Suppressing command output (stdout/stderr) — `@` only suppresses the echoed command text, not the command's own output
- Colorized or formatted echo output — plain text echo is sufficient for v1
- Per-command (multi-line block) suppression via a single `@` — each line is independent unless joined by continuation

---

## 2. Functional Requirements

### FR-1: Default Command Echoing
By default, each resolved shell line is printed to stderr immediately before execution. This gives users visibility into what `ops` is running, similar to Make's default behavior. Echo output goes to stderr so it does not interfere with command stdout (important for piping).

### FR-2: @ Prefix Suppression
When an Opsfile command line begins with `@`, that line's echo is suppressed during execution. The `@` character is stripped before the line is passed to the shell — it is Opsfile syntax, not shell syntax. The command's own output (stdout/stderr) is unaffected.

### FR-3: --silent Global Suppression
When `--silent` (`-s`) is passed, all command echoing is suppressed regardless of whether lines have `@` prefixes. This provides a clean output mode for scripted/automated usage.

### FR-4: --dry-run Compatibility
`--dry-run` continues to print all resolved command lines to stdout (with `@` already stripped). The `@` prefix does not affect dry-run output. When both `--dry-run` and `--silent` are set, nothing is printed (existing behavior preserved).

### FR-5: Multi-Line Command Handling
For backslash-continuation lines (`\` at end of line), the parser joins fragments into a single line before the resolver sees them. If the first fragment starts with `@`, the joined line starts with `@` and the entire joined command is treated as silent. Each independent line in a multi-line command block is evaluated separately for `@`.

An `@` appearing on a non-first continuation fragment (e.g., `aws logs \` / `@--follow`) is not treated as Opsfile syntax — it becomes part of the joined shell text (e.g., `aws logs @--follow`), since `@` detection only applies to the leading character of the final joined line.

### FR-6: @ Detection on Trimmed Lines
The parser applies `strings.TrimSpace()` to every raw Opsfile line before collecting it as a shell line. This means indentation is already stripped before the resolver sees the line, and `@` detection operates on the trimmed result. A line like `    @echo hello` in the Opsfile will have its indentation removed by the parser, producing `@echo hello`, which the resolver correctly detects as a silent line.

### Example Usage

Given an Opsfile:
```
REGION=us-east-1

deploy:
    # Deploy the application
    prod:
        @echo "Deploying to production..."
        aws ecs update-service --cluster prod --service app --region $(REGION)
```

```bash
# Default execution — echoes non-@ lines to stderr
$ ops prod deploy
Deploying to production...              # ← stdout from echo (no command echoed due to @)
aws ecs update-service --cluster prod --service app --region us-east-1   # ← echoed to stderr
{...ecs output...}                      # ← stdout from aws cli

# With --silent — no echoing at all
$ ops --silent prod deploy
Deploying to production...              # ← stdout from echo command
{...ecs output...}                      # ← stdout from aws cli

# Dry-run — shows all resolved lines regardless of @
$ ops --dry-run prod deploy
echo "Deploying to production..."
aws ecs update-service --cluster prod --service app --region us-east-1
```

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Performance | Zero measurable overhead — one `fmt.Fprintln` per non-silent line | Negligible |
| NFR-2 | Compatibility | Works on Linux, macOS; no platform-specific behavior | Echo uses standard stderr |
| NFR-3 | Reliability | `@` stripping is deterministic; only leading `@` consumed | `@@cmd` → `@cmd` (one `@` stripped) |
| NFR-4 | Backwards Compatibility | Existing Opsfiles without `@` gain echoing — this is a visible behavior change | Matches Make convention; `--silent` restores old quiet behavior. Release notes should document: "Commands are now echoed to stderr before execution (like Make). Use `--silent` or `@` prefix to suppress." |
| NFR-5 | Maintainability | Test coverage for resolver `@` stripping and executor echo logic | Table-driven tests |

---

## 4. Architecture & Implementation Proposal

### Overview
The `@` prefix is stripped during command resolution (before variable substitution) and recorded as per-line metadata in a new `ResolvedLine` type. The executor uses this metadata plus a global silent flag to decide whether to echo each line before running it.

### Component Design

**New Type: `ResolvedLine`** (`internal/command_resolver.go`)
```go
type ResolvedLine struct {
    Text   string
    Silent bool // true when the Opsfile line had a leading @ prefix
}
```

**Modified: `ResolvedCommand`** (`internal/command_resolver.go`)
```go
type ResolvedCommand struct {
    Lines []ResolvedLine  // was []string
}
```

**Modified: `Resolve()`** (`internal/command_resolver.go`)
- Before calling `substituteVars()`, check for and strip leading `@`
- Set `Silent: true` on the `ResolvedLine` if `@` was present

**Modified: `Execute()`** (`internal/executor.go`)
- New signature: `Execute(lines []ResolvedLine, shell string, silent bool, echo io.Writer) error`
- The `echo` parameter is the destination for echoed command text (typically `os.Stderr`). This makes the function testable — tests can pass a `bytes.Buffer` to capture and assert on echo output.
- Before each `cmd.Run()`: if `!silent && !line.Silent`, write `line.Text` to the `echo` writer

### Data Flow
```
Opsfile line: "@echo hello $(VAR)"
        |
        v
Parser: stores as-is in OpsCommand.Environments["prod"] = ["@echo hello $(VAR)"]
        |
        v
Resolver: strips "@" → Silent=true, Text="echo hello $(VAR)"
          substituteVars → Text="echo hello world"
          → ResolvedLine{Text: "echo hello world", Silent: true}
        |
        v
main.go: dry-run? → print line.Text
         execute? → Execute(lines, shell, flags.Silent, os.Stderr)
        |
        v
Executor: Silent=true → skip echo
          exec.Command(shell, "-c", "echo hello world")
```

#### Sequence Diagram
```
main.go
  │
  ├─ Resolve(commandName, env, commands, vars)
  │     │
  │     └─ For each raw line:
  │           ├─ Strip leading "@" → set Silent flag
  │           ├─ substituteVars(strippedLine, env, vars)
  │           └─ → ResolvedLine{Text, Silent}
  │
  ├─ If --dry-run:
  │     └─ Print line.Text for each line (unless --silent)
  │
  └─ Execute(resolved.Lines, shell, flags.Silent, os.Stderr)
        └─ For each line:
              ├─ If !silent && !line.Silent → fmt.Fprintln(echo, line.Text)
              ├─ exec.Command(shell, "-c", line.Text)
              └─ cmd.Run() → stop on first error
```

### Key Design Decisions
- **`ResolvedLine` struct over parallel `[]bool`:** Type-safe, self-documenting, extensible for future per-line metadata without changing signatures again
- **Strip `@` in resolver, not parser or executor:** The `@` is Opsfile syntax (not shell syntax). Stripping before variable substitution means `@$(VAR)` works correctly. Keeping it out of the executor maintains separation of concerns.
- **Echo to stderr, not stdout:** Follows Make convention. Prevents echoed commands from contaminating command output when piping (e.g., `ops prod get-id | xargs ...`)
- **Single `@` consumed per line:** `@@echo` becomes `@echo` (a valid shell command). Matches Make behavior. No recursive stripping.
- **`--silent` passed as executor parameter:** Keeps executor a pure function of its inputs, consistent with existing `shell` parameter pattern
- **`io.Writer` for echo destination:** Rather than hardcoding `os.Stderr` in `Execute()`, the echo writer is injected as a parameter. This keeps the executor testable — unit tests pass a `bytes.Buffer` to capture and verify echo output without OS-level pipe redirection. Production callers pass `os.Stderr`.

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/command_resolver.go` | Modify | Add `ResolvedLine` type, change `ResolvedCommand.Lines` to `[]ResolvedLine`, strip `@` in `Resolve()` |
| `internal/executor.go` | Modify | Update `Execute()` signature to accept `[]ResolvedLine`, `silent bool`, and `io.Writer`; add conditional echo logic |
| `cmd/ops/main.go` | Modify | Update dry-run loop to use `line.Text`, pass `flags.Silent` and `os.Stderr` to `Execute()` |
| `internal/command_resolver_test.go` | Modify | Update assertions for `ResolvedLine` type, add `@` prefix test cases |
| `internal/executor_test.go` | Modify | Update calls for new signature, add echo/silent test cases |
| `docs/feature-command-execution.md` | Modify | Document default echoing, `@` prefix, `--silent` interaction |

---

## 5. Alternatives Considered

### Alternative A: Strip @ in the Parser

**Description:** Detect and strip `@` in `opsfile_parser.go` when collecting shell lines. Store metadata in `OpsCommand` (e.g., parallel `[]bool` or new struct).

**Pros:**
- Earliest possible stripping — downstream code never sees `@`

**Cons:**
- Requires changing `OpsCommand.Environments` type from `map[string][]string` to a more complex type
- Larger blast radius — every consumer of `OpsCommand` must adapt
- Mixes Opsfile syntax concerns into the line-collection state machine

**Why not chosen:** The resolver is the natural transformation boundary. `OpsCommand` represents raw parsed data; `ResolvedCommand` represents execution-ready data. Metadata belongs in the resolved output.

---

### Alternative B: Strip @ in the Executor

**Description:** Pass lines with `@` intact through the resolver. Have the executor check for and strip `@` before running.

**Pros:**
- Minimal type changes — `ResolvedCommand.Lines` stays `[]string`
- Only one file changes significantly

**Cons:**
- Executor must understand Opsfile syntax — breaks separation of concerns
- `@` would pass through variable substitution (harmless but semantically wrong)
- `--dry-run` output would include `@` prefix (confusing — it's not shell syntax)

**Why not chosen:** Violates the clean pipeline design where each stage transforms data for the next. The executor should receive execution-ready lines.

---

### Alternative C: No Default Echoing — @ Only Triggers Explicit Echoing

**Description:** Keep the current silent-by-default behavior. Instead, `@` would mean "also echo this line" (inverse of Make).

**Pros:**
- No behavior change for existing Opsfiles

**Cons:**
- Opposite of Make convention — confusing for users familiar with Make
- Contradicts Issue #6 requirements
- Less useful — the common case is wanting to see what's running

**Why not chosen:** Issue #6 explicitly requests Make-style behavior where `@` suppresses echoing, implying echoing should be the default.

---

## Open Questions
- [ ] Should echoed lines have a visual prefix (e.g., `+ command` like `set -x`, or `$ command`)? Current proposal: no prefix, just the raw command text (matching Make).

---

## 6. Task Breakdown

### Phase 1: Foundation
- [ ] Add `ResolvedLine` struct to `internal/command_resolver.go`
- [ ] Change `ResolvedCommand.Lines` from `[]string` to `[]ResolvedLine`
- [ ] Update `Resolve()` to strip `@` prefix and populate `Silent` field
- [ ] Update existing resolver tests for new type (add `lineTexts()` helper)
- [ ] Add new resolver tests for `@` prefix stripping and edge cases

### Phase 2: Integration
- [ ] Update `Execute()` signature to accept `[]ResolvedLine`, `silent bool`, and `echo io.Writer`
- [ ] Add conditional echo logic in `Execute()` using the injected writer
- [ ] Update `cmd/ops/main.go` dry-run loop and `Execute()` call (pass `os.Stderr` as echo writer)
- [ ] Update existing executor tests for new signature (add `toLines()` helper)
- [ ] Add new executor tests for echo and silent behavior

### Phase 3: Polish
- [ ] Update `docs/feature-command-execution.md` with echoing behavior
- [ ] Manual end-to-end testing with example Opsfiles
- [ ] Run `make lint` and `make test` for final validation

---
