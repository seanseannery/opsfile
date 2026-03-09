# Feature: CLI Argument and Flag Parsing


## 1. Problem Statement & High-Level Goals

### Problem
The `ops` CLI needs to accept user input in two forms: flags (which modify tool behavior) and positional arguments (which select the environment, command, and pass-through args as defined in the Opsfile). Without structured parsing, the tool cannot distinguish between its own operational flags and the user's intended command invocation.

### Goals
- [x] Parse ops-level flags (`--dry-run`, `--silent`, `--directory`, `--version`, `--list`, `--help`) from the command line
- [x] Extract positional arguments into environment, command, and pass-through command args
- [x] Provide clear error messages for unknown flags or missing required arguments

### Non-Goals
- Sub-command style parsing (e.g. `ops run prod tail-logs`) — the CLI uses a flat `ops [flags] <env> <command>` structure
- Parsing or validating Opsfile-defined command arguments — positional args after the command name are passed through as-is

---

## 2. Functional Requirements

### FR-1: Flag Parsing
The CLI accepts the following flags before positional arguments:
- `-D <directory>` / `--directory <directory>` — use the Opsfile in the given directory instead of discovering it
- `-d` / `--dry-run` — print resolved commands without executing them
- `-s` / `--silent` — suppress output during execution (also suppresses dry-run output)
- `-l` / `--list` — list available commands and environments from the Opsfile
- `-v` / `--version` — print the ops version string and exit
- `-h` / `--help` / `-?` — print usage text and exit

Unknown flags produce an error. Flag parsing stops at the first non-flag argument (interspersed mode disabled), so all remaining tokens are treated as positionals.

### FR-2: Help Flag Handling
All help tokens (`-h`, `--help`, `-?`) are stripped from the argument list before parsing so that flags appearing after the help flag are still parsed (e.g. `--help -D /path` parses `-D` correctly). After parsing, if help was requested, the usage banner is printed and `ErrHelp` is returned.

### FR-3: Positional Argument Parsing
After flags are consumed, the remaining positional arguments follow the structure: `<environment> <command> [command-args...]`
- The first positional argument is the environment name (e.g. `prod`, `preprod`, `local`)
- The second positional argument is the command name (e.g. `tail-logs`, `list-instance-ips`)
- Any additional positional arguments are passthrough command args, preserved in order
- Missing environment produces an error: `"missing environment argument"`
- Missing command (only one positional) produces an error: `"missing command argument"`

### Example Usage

```bash
# Basic invocation
ops prod tail-logs

# With flags
ops --dry-run prod tail-logs
ops -D /path/to/project prod deploy

# Help
ops --help
ops -?

# List commands
ops --list
ops -l

# With pass-through args
ops prod search-logs "error" "--since=1h"
```

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Compatibility | Runs on Linux, macOS, and Windows | Cross-platform Go binary |
| NFR-2 | Reliability | Clear error messages for invalid flags and missing arguments | Uses `slog.Error` + non-zero exit |
| NFR-3 | Maintainability | Test coverage for flag and argument parsing edge cases | Table-driven tests in `flag_parser_test.go` |
| NFR-4 | Usability | Short and long flag forms for all flags | e.g. `-d` / `--dry-run` |

---

## 4. Architecture & Implementation Proposal

### Overview
Flag parsing uses `github.com/spf13/pflag` (POSIX-compatible flag library) with interspersed mode disabled so that flag parsing stops at the first positional argument. Positional argument parsing is a simple slice-index extraction.

### Component Design
All parsing logic lives in `internal/flag_parser.go` and exposes two public functions:
- `ParseOpsFlags` — handles flag extraction and returns remaining positionals
- `ParseOpsArgs` — splits positionals into environment, command, and command args

### Data Flow
```
os.Args[1:] -> ParseOpsFlags() -> (OpsFlags, positionals, error)
                                       |
                                positionals -> ParseOpsArgs() -> (Args, error)
```

#### Sequence Diagram
```
User CLI Input
     │
     ▼
ParseOpsFlags(osArgs, usageOutput)
     │
     ├─ Strip help tokens (-h, --help, -?)
     ├─ pflag.Parse(filtered args)
     ├─ If help requested → print usage, return ErrHelp
     └─ Return OpsFlags + remaining positionals
              │
              ▼
     ParseOpsArgs(positionals)
              │
              ├─ Validate len >= 2
              └─ Return Args{OpsEnv, OpsCommand, CommandArgs}
```

### Key Design Decisions
- **pflag over stdlib `flag`:** pflag provides POSIX-style `--long` and `-short` flags, `SetInterspersed(false)` to stop parsing at first positional, and `StringP`/`BoolP` helpers for dual short/long registration
- **Help token stripping:** Help flags are manually stripped before `pflag.Parse` so that flags appearing after `--help` are still parsed (e.g. `--help -D /path` correctly populates `Directory`)
- **`-?` manual handling:** `-?` is not a valid pflag name, so it is detected during the stripping phase alongside `-h` and `--help`
- **Separate Parse functions:** Flag parsing and positional parsing are separate functions to keep concerns isolated and allow `--list` and `--version` to short-circuit before positional validation

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/flag_parser.go` | Exists | Flag and argument parsing logic |
| `internal/flag_parser_test.go` | Exists | Tests for parsing edge cases |
| `cmd/ops/main.go` | Exists | Wires `ParseOpsFlags` and `ParseOpsArgs` into the CLI pipeline |

---

## 5. Alternatives Considered

### Alternative A: Standard Library `flag` Package

**Description:** Use Go's built-in `flag` package instead of `spf13/pflag`.

**Pros:**
- Zero external dependencies
- Part of the standard library

**Cons:**
- No native `--long-flag` support (uses `-flag` for both short and long)
- No `SetInterspersed(false)` — requires manual workarounds to stop flag parsing at positionals
- Requires sharing variable pointers to alias short/long forms

**Why not chosen:** pflag provides a cleaner API for POSIX-style flags with minimal dependency cost and active community maintenance. The original implementation used stdlib `flag` but was migrated to pflag.

---

## Open Questions
- [x] ~~Should `CommandArgs` be plumbed through to execution?~~ Field exists and is parsed; plumbing is a separate concern handled downstream.

---

## 6. Task Breakdown

*This feature is fully implemented. Tasks listed retrospectively.*

### Phase 1: Foundation
- [x] Define `OpsFlags` and `Args` types
- [x] Implement `ParseOpsFlags` with short/long flag registration
- [x] Implement `ParseOpsArgs` with positional extraction
- [x] Write unit tests for flag and argument parsing

### Phase 2: Integration
- [x] Wire `ParseOpsFlags` and `ParseOpsArgs` into `cmd/ops/main.go`
- [x] Handle `ErrHelp` and `--version` short-circuits in main
- [x] Add `--list` flag for command listing

### Phase 3: Polish
- [x] Add `-?` help alias support
- [x] Ensure help flag doesn't prevent other flags from being parsed
- [x] Migrate from stdlib `flag` to `spf13/pflag` for POSIX compatibility

---
