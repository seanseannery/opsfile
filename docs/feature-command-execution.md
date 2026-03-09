# Feature: Command Execution


## 1. Problem Statement & High-Level Goals

### Problem
After the user specifies an environment and command via the CLI, `ops` must map that input to the correct set of shell lines defined in the Opsfile, substitute any variable references, and execute the resulting commands in the user's shell. Without this, the Opsfile is just a static document with no runtime behavior.

### Goals
- [x] Resolve the correct environment-specific command block from the parsed Opsfile, with fallback to `default`
- [x] Substitute `$(VAR_NAME)` references using the four-level variable priority chain
- [x] Execute resolved shell lines sequentially in the user's shell with full terminal interactivity
- [x] Support `--dry-run` to print resolved commands without executing and `--silent` to suppress output

### Non-Goals
- Parallel command execution — lines are always run sequentially
- Built-in retry or rollback logic — the tool runs commands as-is
- Remote execution — all commands run locally via the user's shell

---

## 2. Functional Requirements

### FR-1: Command Resolution
- Given a command name and environment, the resolver selects the correct set of shell lines to execute
- Environment selection follows this priority:
  1. Exact match on the requested environment (e.g. `prod`)
  2. Fallback to the `default` environment block if the specific one is absent
  3. Error if neither the specific environment nor `default` exists
- If the command name does not exist in the parsed commands map, an error is returned

### FR-2: Variable Substitution
- After environment selection, all `$(VAR_NAME)` tokens in the shell lines are substituted using the four-level variable priority chain (see feature-variable-substitution.md for details)
- Non-identifier tokens inside `$(...)` (e.g. shell subcommands like `$(shell ...)`) are passed through unchanged
- Unclosed `$(` without a matching `)` is treated as a literal string

### FR-3: Shell Execution
- Resolved shell lines are executed sequentially, one at a time
- Each line is run via the user's `$SHELL` environment variable; if unset, `/bin/sh` is used as the fallback
- Each command is invoked as `<shell> -c <line>`
- Commands inherit the current process environment, stdin, stdout, and stderr (fully interactive)
- Execution stops immediately on the first command that returns a non-zero exit code
- The exit code from a failed command is propagated as the exit code of the `ops` process itself
- If no shell lines are provided (empty list), execution is a no-op

### FR-4: Dry-Run and Silent Modes
- When `--dry-run` is set, resolved commands are printed to stdout instead of being executed
- When both `--dry-run` and `--silent` are set, nothing is printed and no commands are executed
- Dry-run and silent flag handling is implemented in `cmd/ops/main.go`, not in the executor itself

### Example Usage

Given an Opsfile:
```
[vars]
CLUSTER=my-cluster

[command.tail-logs]
prod: kubectl logs -f deployment/app --context=$(CLUSTER)
default: echo "use 'ops prod tail-logs' for production logs"
```

```bash
# Executes the prod block with variable substitution
ops prod tail-logs
# → runs: kubectl logs -f deployment/app --context=my-cluster

# Falls back to default block
ops local tail-logs
# → runs: echo "use 'ops prod tail-logs' for production logs"

# Dry-run: prints resolved command without executing
ops --dry-run prod tail-logs
# → prints: kubectl logs -f deployment/app --context=my-cluster
```

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Performance | Minimal overhead — execution latency dominated by the shell commands themselves | No batching or pooling |
| NFR-2 | Compatibility | Uses `$SHELL` with `/bin/sh` fallback; works on Linux, macOS, and Windows | Shell selection in `main.go` |
| NFR-3 | Reliability | Fail-fast on first non-zero exit code with error propagation | Exit code forwarded to OS |
| NFR-4 | Interactivity | Full stdin/stdout/stderr passthrough for interactive commands | e.g. `ssh`, `vim`, `less` |
| NFR-5 | Maintainability | Resolver and executor are separate packages with distinct responsibilities | Testable in isolation |

---

## 4. Architecture & Implementation Proposal

### Overview
Command execution is split into two stages: resolution (selecting + substituting) and execution (running in shell). The resolver lives in `internal/command_resolver.go` and the executor in `internal/executor.go`. Dry-run/silent logic is handled at the call site in `main.go`.

### Component Design

**Command Resolver (`internal/command_resolver.go`):**
- `Resolve()` — orchestrates command lookup, environment selection, and variable substitution
- `selectLines()` — implements the env-then-default fallback logic
- `substituteVars()` — replaces `$(VAR_NAME)` tokens with resolved values
- `resolveVar()` — implements the four-level priority chain lookup

**Executor (`internal/executor.go`):**
- `Execute()` — runs shell lines sequentially via `exec.Command`

### Data Flow
```
(commandName, env, commands, vars) -> Resolve() -> ResolvedCommand{Lines}
                                                          |
                                              main.go: dry-run check
                                                          |
                                                 Execute(lines, shell)
                                                          |
                                                  exec.Command(shell, "-c", line)
```

#### Sequence Diagram
```
main.go
  │
  ├─ Resolve(commandName, env, commands, vars)
  │     │
  │     ├─ Lookup command in commands map
  │     ├─ selectLines(cmd, env)
  │     │     ├─ Try cmd.Environments[env]
  │     │     └─ Fallback to cmd.Environments["default"]
  │     └─ substituteVars(line, env, vars) for each line
  │           └─ resolveVar(token, env, vars)
  │                 ├─ 1. vars[env+"_"+VAR]
  │                 ├─ 2. os.LookupEnv(env+"_"+VAR)
  │                 ├─ 3. vars[VAR]
  │                 └─ 4. os.LookupEnv(VAR)
  │
  ├─ If --dry-run: print lines (unless --silent), return
  │
  └─ Execute(lines, shell)
        └─ For each line:
              ├─ exec.Command(shell, "-c", line)
              ├─ Wire stdin/stdout/stderr
              └─ cmd.Run() — stop on first error
```

### Key Design Decisions
- **Separate resolver and executor:** Keeps variable substitution testable without shell execution, and allows dry-run to operate on resolved output without touching the executor
- **Fail-fast execution:** Stops on first non-zero exit code rather than continuing, matching `set -e` shell behavior and preventing cascading failures
- **Shell selection in main.go:** The executor receives the shell as a parameter rather than looking it up itself, keeping it a pure function of its inputs
- **Pass-through for non-identifier `$(...)` tokens:** Preserves shell subcommand syntax like `$(date +%s)` while substituting Opsfile variables

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/command_resolver.go` | Exists | Command lookup, environment selection, variable substitution |
| `internal/executor.go` | Exists | Sequential shell line execution |
| `internal/command_resolver_test.go` | Exists | Tests for resolution, env fallback, and substitution |
| `internal/executor_test.go` | Exists | Tests for execution behavior |
| `cmd/ops/main.go` | Exists | Wires resolver and executor; handles dry-run/silent logic |

---

## 5. Alternatives Considered

### Alternative A: Single Execute Function with Embedded Resolution

**Description:** Combine resolution and execution into one function that takes raw command name, env, and vars.

**Pros:**
- Simpler call site — one function call instead of two

**Cons:**
- Harder to test resolution logic independently
- No clean way to support dry-run without executing
- Violates single-responsibility principle

**Why not chosen:** Separating resolution from execution enables dry-run mode, isolated testing, and clearer code organization.

---

### Alternative B: Shell Script Generation

**Description:** Generate a temporary shell script from resolved lines and execute it as a single file.

**Pros:**
- Single process invocation
- Could support shell features like `set -e` natively

**Cons:**
- Temp file management and cleanup
- Harder to attribute errors to specific lines
- Security concerns with temp file permissions

**Why not chosen:** Per-line execution is simpler, provides better error attribution, and avoids temp file management.

---

## Open Questions
- (none currently)

---

## 6. Task Breakdown

*This feature is fully implemented. Tasks listed retrospectively.*

### Phase 1: Foundation
- [x] Define `ResolvedCommand` type
- [x] Implement `selectLines` with env-then-default fallback
- [x] Implement `substituteVars` with identifier detection and pass-through for non-identifiers
- [x] Implement `resolveVar` with four-level priority chain
- [x] Write unit tests for command resolution

### Phase 2: Integration
- [x] Implement `Execute` function with sequential shell line execution
- [x] Wire resolver and executor into `main.go`
- [x] Add shell selection logic (`$SHELL` with `/bin/sh` fallback)
- [x] Add exit code propagation via `errors.As` + `exec.ExitError`

### Phase 3: Polish
- [x] Add dry-run support (print resolved lines without executing)
- [x] Add silent mode support (suppress dry-run output)
- [x] Write executor tests

---
