# Feature: `-` Prefix to Ignore Non-Zero Exit Codes (Issue #20)


## 1. Problem Statement & High-Level Goals

### Problem
Currently, `ops` stops execution on the first shell line that returns a non-zero exit code (fail-fast behavior). This is appropriate for most workflows, but some operations include lines that may legitimately fail without indicating a real problem (e.g., `rm -f`, `docker stop` on a container that may not be running, `killall` on a process that may not exist). Users have no way to express "ignore failure on this line and continue" within an Opsfile. In Makefiles, the `-` prefix serves this purpose. (Issue #20)

### Goals
- [ ] Support `-` prefix on Opsfile shell lines to ignore non-zero exit codes and continue execution
- [ ] For multi-line commands (backslash continuation and indent continuation), `-` on the first line applies to the entire joined command
- [ ] Support combined `-@` and `@-` prefixes in either order (order-independent)
- [ ] Update example Opsfile with `-` prefix usage
- [ ] Add unit and integration tests

### Non-Goals
- Suppressing stderr output from the failing command -- `-` only ignores the exit code, it does not hide output
- Applying `-` to an entire command block (all lines) via a single annotation -- each line is independent
- Retry logic or conditional execution based on exit codes -- `-` is binary: ignore or fail

---

## 2. Functional Requirements

### FR-1: Dash Prefix Ignores Non-Zero Exit Codes
When an Opsfile command line begins with `-`, any non-zero exit code from that line is ignored and execution proceeds to the next line. The `-` character is stripped before the line is passed to the shell -- it is Opsfile syntax, not shell syntax. The command's stdout and stderr output are unaffected.

### FR-2: Multi-Line Command Inheritance
For backslash-continuation lines (`\` at end of line) and indent-continuation lines, the parser joins fragments into a single line before the resolver sees them. If the first fragment starts with `-`, the entire joined command inherits the ignore-failure behavior. The `-` does not need to appear on subsequent continuation fragments. A `-` appearing on a non-first continuation fragment is part of the joined shell text, not Opsfile syntax (same behavior as `@`).

### FR-3: Combined Prefix Support (`-@` and `@-`)
Both `-@` and `@-` are valid and equivalent. When both prefixes are present, the line has its echo suppressed (`@` behavior) AND its non-zero exit code ignored (`-` behavior). The resolver strips both prefix characters regardless of order before passing the remainder to variable substitution.

### FR-4: Dry-Run Interaction
When `--dry-run` is set, all resolved command lines are printed to stdout (with `-` and `@` already stripped). The `-` prefix does not affect dry-run output. Dry-run should show the command as it would be executed, without prefix syntax.

### FR-5: Single Prefix Consumed Per Character
`--echo hello` becomes `-echo hello` (one `-` stripped, `IgnoreError: true`). This mirrors how `@@echo` becomes `@echo` (one `@` stripped). Only one leading `-` and one leading `@` are consumed as Opsfile syntax; anything remaining is shell text.

### FR-6: Dash in Middle of Line
A `-` character appearing anywhere other than the leading position of a resolved line is not treated as Opsfile syntax. For example, `kubectl delete --force` or `echo "hello-world"` are unaffected. This is consistent with how `@` is only significant at the leading position.

### Example Usage

Given an Opsfile:
```
cleanup:
    default:
        -docker stop my-app
        -docker rm my-app
        docker run -d --name my-app my-image
        @echo "Deployment complete"

teardown:
    prod:
        -@kubectl delete pod old-pod
        -killall background-worker
        echo "Teardown finished"
```

```bash
# Normal execution -- docker stop/rm may fail if container doesn't exist, but execution continues
$ ops default cleanup
docker stop my-app                    # echoed to stderr
Error response from daemon: ...       # stderr from docker (container not running -- ignored)
docker rm my-app                      # echoed to stderr
Error: No such container: my-app      # stderr from docker -- ignored
docker run -d --name my-app my-image  # echoed to stderr
abc123def456                          # stdout from docker run
Deployment complete                   # stdout from echo (not echoed due to @)

# Dry-run -- shows resolved commands without prefixes
$ ops --dry-run default cleanup
docker stop my-app
docker rm my-app
docker run -d --name my-app my-image
echo "Deployment complete"

# Combined prefix -- both @ and - applied
$ ops prod teardown
                                      # kubectl line: not echoed (@), exit code ignored (-)
killall background-worker             # echoed to stderr, exit code ignored (-)
echo "Teardown finished"              # echoed to stderr
Teardown finished                     # stdout from echo
```

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Performance | Zero measurable overhead -- one conditional check per line | Negligible |
| NFR-2 | Compatibility | Works on Linux and macOS; no platform-specific behavior | Uses existing exec.Command path |
| NFR-3 | Reliability | Prefix stripping is deterministic; only one leading `-` consumed | `--cmd` becomes `-cmd` |
| NFR-4 | Backwards Compatibility | No behavior change for existing Opsfiles -- `-` prefix is opt-in | Existing fail-fast behavior unchanged |
| NFR-5 | Maintainability | Test coverage for resolver `-` stripping, combined prefixes, and executor ignore-error logic | Table-driven tests |

---

## 4. Architecture & Implementation Proposal

### Overview
The `-` prefix follows the same architectural pattern established by the `@` prefix (Issue #6). It is detected and stripped during command resolution in `Resolve()`, recorded as per-line metadata in the existing `ResolvedLine` struct, and acted upon during execution in `Execute()`. This keeps the parser, resolver, and executor responsibilities clean and consistent.

### Component Design

**Modified: `ResolvedLine`** (`internal/command_resolver.go`)
```go
type ResolvedLine struct {
    Text        string
    Silent      bool // true when the Opsfile line had a leading @ prefix
    IgnoreError bool // true when the Opsfile line had a leading - prefix
}
```

**Modified: `Resolve()`** (`internal/command_resolver.go`)

The current prefix-stripping logic handles only `@`:
```go
if strings.HasPrefix(line, "@") {
    silent = true
    line = line[1:]
}
```

This must be replaced with a loop that handles both `@` and `-` in any order, consuming at most one of each:
```go
silent := false
ignoreError := false
for len(line) > 0 {
    switch line[0] {
    case '@':
        if !silent {
            silent = true
            line = line[1:]
            continue
        }
    case '-':
        if !ignoreError {
            ignoreError = true
            line = line[1:]
            continue
        }
    }
    break
}
```

This approach:
- Handles `@-`, `-@`, `@`, `-`, and bare lines uniformly
- Consumes at most one `@` and one `-` (so `@@` strips one `@`, `--` strips one `-`)
- Preserves the remaining line text for variable substitution
- Runs in O(1) since at most two iterations occur

**Modified: `Execute()`** (`internal/executor.go`)

The current error handling returns immediately on failure:
```go
if err := cmd.Run(); err != nil {
    return fmt.Errorf("running %q: %w", line.Text, err)
}
```

When `line.IgnoreError` is true, only `*exec.ExitError` (non-zero exit code) is ignored — system-level errors such as shell-not-found or permission-denied still propagate. This matches Make's `-` behavior, which only suppresses exit code failures, not execution failures:
```go
if err := cmd.Run(); err != nil {
    var exitErr *exec.ExitError
    if line.IgnoreError && errors.As(err, &exitErr) {
        continue // non-zero exit code ignored per - prefix
    }
    return fmt.Errorf("running %q: %w", line.Text, err)
}
```

No changes to echo logic -- `Silent` and `IgnoreError` are orthogonal concerns.

### Data Flow
```
Opsfile line: "-@docker stop old-app"
        |
        v
Parser: stores as-is in OpsCommand.Environments["default"] = ["-@docker stop old-app"]
        |
        v
Resolver: strips "-" -> IgnoreError=true
          strips "@" -> Silent=true
          remaining text: "docker stop old-app"
          substituteVars -> Text="docker stop old-app"
          -> ResolvedLine{Text: "docker stop old-app", Silent: true, IgnoreError: true}
        |
        v
main.go: dry-run? -> print line.Text (no prefixes shown)
         execute? -> Execute(lines, shell, flags.Silent, os.Stderr)
        |
        v
Executor: Silent=true -> skip echo
          exec.Command(shell, "-c", "docker stop old-app")
          cmd.Run() returns exit code 1
          IgnoreError=true -> discard error, continue to next line
```

#### Sequence Diagram
```
main.go
  |
  +-- Resolve(commandName, env, commands, vars)
  |     |
  |     +-- For each raw line:
  |           +-- Strip leading "@" -> set Silent flag (at most once)
  |           +-- Strip leading "-" -> set IgnoreError flag (at most once)
  |           +-- substituteVars(strippedLine, env, vars)
  |           +-- -> ResolvedLine{Text, Silent, IgnoreError}
  |
  +-- If --dry-run:
  |     +-- Print line.Text for each line (unless --silent)
  |
  +-- Execute(resolved.Lines, shell, flags.Silent, os.Stderr)
        +-- For each line:
              +-- If !silent && !line.Silent -> fmt.Fprintln(echo, line.Text)
              +-- exec.Command(shell, "-c", line.Text)
              +-- cmd.Run()
              +-- If err != nil && !line.IgnoreError -> return error
              +-- If err != nil && line.IgnoreError -> continue
```

### Key Design Decisions

- **Add `IgnoreError` field to existing `ResolvedLine` rather than a new type:** The `ResolvedLine` struct was designed to hold per-line metadata (Issue #6 added `Silent`). Adding `IgnoreError` is the natural extension. No new types needed.
- **Strip `-` in resolver, not parser or executor:** Consistent with the `@` prefix architecture. The resolver is the transformation boundary between raw Opsfile syntax and execution-ready commands. The parser stores raw text; the executor receives clean commands with metadata.
- **Loop-based prefix stripping over sequential if-checks:** A loop naturally handles any ordering of `@` and `-` without duplicating logic for each permutation. The `if !silent` / `if !ignoreError` guards ensure at most one of each is consumed.
- **`IgnoreError` is orthogonal to `Silent`:** They are independent concerns. A line can be silent, ignore errors, both, or neither. No interaction between them in the executor beyond both being fields on the same struct.
- **Dry-run shows the command without prefixes:** Consistent with `@` behavior. The user sees the actual shell command that would execute, not Opsfile syntax.

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/command_resolver.go` | Modify | Add `IgnoreError` field to `ResolvedLine`; refactor prefix stripping in `Resolve()` to handle both `@` and `-` in any order |
| `internal/executor.go` | Modify | Add `IgnoreError` check in the error handling path of `Execute()` |
| `internal/command_resolver_test.go` | Modify | Add test cases for `-` prefix stripping, combined `-@`/`@-` prefixes, double-dash, dash in middle of line, multi-line continuation with dash |
| `internal/executor_test.go` | Modify | Add test cases for `IgnoreError` behavior: ignored exit codes, combined with `Silent`, error propagation when `IgnoreError=false` |
| `examples/*.Opsfile` | Modify | Add example command demonstrating `-` prefix usage |
| `docs/feature-command-execution.md` | Modify | Document `-` prefix behavior and interaction with `@` and `--dry-run` |
| `docs/feature-at-prefix-suppress.md` | Modify | Add note about combined `-@`/`@-` prefix support |

---

## 5. Alternatives Considered

### Alternative A: Handle `-` Prefix in the Executor Only

**Description:** Pass the `-` prefix through the resolver untouched. Have the executor check for and strip `-` before running the command.

**Pros:**
- Minimal changes to resolver -- only executor changes
- No new fields on `ResolvedLine`

**Cons:**
- Executor must understand Opsfile syntax -- breaks separation of concerns
- `--dry-run` output would include `-` prefix (confusing for users)
- Inconsistent with how `@` prefix is handled (stripped in resolver)
- Combined prefix handling becomes messy -- executor needs to know about both `@` and `-`

**Why not chosen:** Violates the established architecture. The `@` prefix set the precedent: Opsfile syntax is resolved in the resolver, not the executor.

---

### Alternative B: Shell-Level Error Suppression

**Description:** Instead of stripping `-` and handling it in Go, prepend the command with `|| true` or wrap it in a subshell that ignores errors.

**Pros:**
- No changes to executor error handling
- Leverages shell behavior directly

**Cons:**
- Modifies the command text -- changes what gets echoed and what appears in dry-run
- `|| true` changes the exit code to 0, which may mask errors in unexpected ways
- Subshell wrapping changes the execution context (e.g., variable scope, directory)
- Not consistent with Make's approach (which handles `-` at the runner level, not shell level)

**Why not chosen:** Altering command text is surprising to users and creates subtle behavioral differences. The Go-level approach is cleaner and matches Make's implementation strategy.

---

### Alternative C: New Field on `OpsCommand` (Parser-Level)

**Description:** Detect `-` during parsing in `opsfile_parser.go` and store it as metadata on the parsed command lines, similar to how an AST might work.

**Pros:**
- Earliest possible detection of prefix

**Cons:**
- Requires changing `OpsCommand.Environments` from `map[string][]string` to a more complex type
- Large blast radius -- every consumer of `OpsCommand` must adapt
- Inconsistent with how `@` was implemented (resolver, not parser)
- The parser's job is to collect raw lines; semantic interpretation belongs in the resolver

**Why not chosen:** Same reasoning as Alternative A in the `@` prefix design doc. The parser stores raw text; the resolver interprets syntax.

---

## Open Questions
- [x] Should stderr from an ignored-error line be visually differentiated (e.g., dimmed or prefixed with a warning)? **No** — stderr passes through unchanged, matching Make behavior.
- [x] What happens with `-@-` or `@-@`? The loop consumes at most one `-` and one `@`. `-@-` strips one `-` and one `@`, leaving `-` as shell text. `@-@` strips one `@` and one `-`, leaving `@` as shell text. Both are correct and deterministic; a test case covers this.
- [x] How does `-` prefix interact with `CommandArgs` passthrough? `CommandArgs` are appended at the shell invocation level after the resolver runs. Since `-` is stripped in the resolver and `IgnoreError` is metadata on `ResolvedLine`, there is no interaction. A test case validates this.
- [x] Should system-level errors (shell not found, permission denied) be suppressed by `-`? **No** — only `*exec.ExitError` is ignored. `errors.As(err, &exitErr)` ensures non-exit errors always propagate.

---

## 6. Task Breakdown

### Phase 1: Foundation
- [ ] Add `IgnoreError bool` field to `ResolvedLine` in `internal/command_resolver.go`
- [ ] Refactor prefix stripping in `Resolve()` from single `@` check to loop handling both `@` and `-`
- [ ] Write resolver unit tests for `-` prefix: stripping, combined `-@`/`@-`, double-dash, dash-only, dash in middle of line
- [ ] Write resolver unit tests for multi-line continuation with `-` prefix (backslash and indent)

### Phase 2: Integration
- [ ] Add `IgnoreError` check to `Execute()` error path in `internal/executor.go`
- [ ] Write executor unit tests: ignored exit code continues, non-ignored exit code still fails, combined `Silent`+`IgnoreError`, `IgnoreError` with invalid shell
- [ ] Verify `--dry-run` output strips `-` prefix (covered by existing dry-run path since resolver already strips it)

### Phase 3: Polish
- [ ] Add `-` prefix example to an example Opsfile in `examples/`
- [ ] Update `docs/feature-command-execution.md` with `-` prefix behavior
- [ ] Update `docs/feature-at-prefix-suppress.md` with combined prefix note
- [ ] Run `make lint` and `make test` for final validation

---
