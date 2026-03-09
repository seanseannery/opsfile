# Feature: Supported CLI Flags


## 1. Problem Statement & High-Level Goals

### Problem
The `ops` CLI needs to support standard operational flags that modify its behavior — such as previewing commands without execution, suppressing output, printing version info, listing available commands, specifying an alternate Opsfile directory, and displaying usage help. Without these flags, users have no way to inspect, debug, or control `ops` behavior beyond running commands directly.

### Goals
- [x] Provide `--dry-run` / `-d` to preview resolved commands without executing them
- [x] Provide `--silent` / `-s` to suppress command output during execution
- [x] Provide `--version` / `-v` to display the current ops version and build info
- [x] Provide `--help` / `-h` / `-?` to display usage information
- [x] Provide `--directory` / `-D` to specify an alternate Opsfile location
- [x] Provide `--list` / `-l` to list available commands and environments from the Opsfile

### Non-Goals
- Flags are not subcommand-specific; they apply globally to `ops`
- No interactive/prompt-based flag input
- No configuration file for default flag values

---

## 2. Functional Requirements

### FR-1: Dry-Run Mode (`--dry-run` / `-d`)
When `--dry-run` or `-d` is passed, `ops` prints the fully resolved shell lines to stdout without executing them. Each line is printed on its own line via `fmt.Println`. Variable substitution and environment selection still occur — the user sees the exact commands that *would* be executed. If `--silent` is also set, even the dry-run output is suppressed (nothing is printed and nothing is executed).

### FR-2: Silent Mode (`--silent` / `-s`)
When `--silent` or `-s` is passed, command execution proceeds normally but stdout/stderr output is not explicitly suppressed at the `ops` level — the executor connects stdin/stdout/stderr to the terminal. Silent mode's primary effect is suppressing dry-run output when combined with `--dry-run`.

### FR-3: Version Reporting (`--version` / `-v`)
When `-v` or `--version` is passed, `ops` prints its version string and exits with code 0. The output format is: `ops version <version> (commit: <commit>) <os>/<arch>` (e.g. `ops version 0.8.5 (commit: abc1234) darwin/arm64`). The version and commit are set at build time via Go linker flags; defaults are `0.0.0-dev` and `none`.

### FR-4: Help / Usage Output (`--help` / `-h` / `-?`)
When `-h`, `--help`, or `-?` is passed, `ops` prints a usage banner and flag descriptions to stderr, then exits with code 0. All help tokens are stripped from args before parsing so that flags appearing after the help flag (e.g. `--help -D /path`) are still parsed. The usage banner includes the version info, a description of what `ops` does, invocation syntax, an example, and the full list of available flags via `pflag.PrintDefaults()`. When an Opsfile is discoverable, help output also appends a listing of available commands and environments (best-effort; Opsfile errors are silently ignored).

### FR-5: Directory Override (`--directory` / `-D`)
When `-D <dir>` or `--directory <dir>` is passed, `ops` uses the Opsfile from the specified directory instead of walking parent directories. This interacts with all other flags — `--list` and `--help` will use the specified directory's Opsfile.

### FR-6: List Commands (`--list` / `-l`)
When `--list` or `-l` is passed, `ops` prints a summary of the Opsfile's available environments and commands, then exits with code 0. No command resolution or execution occurs — even if an environment and command are also supplied on the CLI. See the List Commands section below for full details.

### Example Usage

**Dry-run preview:**
```bash
$ ops --dry-run prod tail-logs
aws logs tail /ecs/prod-app --follow --since 5m
```

**Version output:**
```bash
$ ops --version
ops version 0.8.5 (commit: abc1234) darwin/arm64
```

**Help output:**
```bash
$ ops --help
ops version 0.8.5 (commit: abc1234) darwin/arm64

The 'ops' command runs commonly-used live-operation commands...

Usage: ops [flags] <environment> <command> [command-args]
      ex. 'ops preprod open-dashboard' or 'ops --dry-run prod tail-logs'

Flags:
  -D, --directory directory   use Opsfile in the given directory
  -d, --dry-run               print commands without executing
  -l, --list                  list available commands and environments
  -s, --silent                execute without printing output
  -v, --version               print the ops version and exit
```

**List output:**
```bash
$ ops --list

Commands Found in [./Opsfile]:

Environments:
  default  local  preprod  prod

Commands:
  show-profile       Using AWS profile
  tail-logs          Tail CloudWatch logs
  list-instance-ips  List the private IPs of running instances
```

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Performance | Flag parsing adds negligible overhead (< 1ms) | Uses pflag which is standard in Go CLIs |
| NFR-2 | Compatibility | All flags work on Linux, macOS, and Windows | Uses Go standard library + pflag |
| NFR-3 | Reliability | Unknown flags produce a clear error and exit non-zero | pflag returns an error for unrecognised flags |
| NFR-4 | Maintainability | New flags only require adding to `OpsFlags` struct and registering with pflag | Centralized in `flag_parser.go` |

---

## 4. Architecture & Implementation Proposal

### Overview
Flag parsing is centralized in `internal/flag_parser.go` using the `pflag` library. The `ParseOpsFlags` function parses all ops-level flags from the raw CLI arguments and returns an `OpsFlags` struct plus remaining positional arguments. `cmd/ops/main.go` inspects the flags struct to determine which early-exit path to take (help, version, list) before proceeding to command resolution and execution.

### Component Design

**`internal/flag_parser.go`** — Defines `OpsFlags` struct, `ErrHelp` sentinel, `ParseOpsFlags()`, and `ParseOpsArgs()`. Uses `pflag.NewFlagSet` with `ContinueOnError` error handling and `SetInterspersed(false)` to stop flag parsing at the first positional argument. Help tokens (`-h`, `--help`, `-?`) are stripped before parsing to ensure all other flags are still processed.

**`internal/version.go`** — Declares `Version` and `Commit` variables with defaults (`0.0.0-dev` / `none`), overridden at build time via `-ldflags`.

**`internal/lister.go`** — `FormatCommandList()` writes a formatted summary of environments and commands to an `io.Writer`, enabling testability without touching stdout.

**`cmd/ops/main.go`** — Sequences the flag checks: help → version → parse Opsfile → list → parse args → resolve → dry-run check → execute.

### Data Flow
```
os.Args -> ParseOpsFlags() -> OpsFlags + positionals
  |
  ├─ ErrHelp? -> print usage + best-effort command listing -> exit 0
  ├─ Version?  -> print version string -> exit 0
  ├─ List?     -> ParseOpsFile -> FormatCommandList -> exit 0
  └─ otherwise -> ParseOpsArgs -> Resolve -> DryRun check -> Execute
```

#### Sequence Diagram
```
User -> main.go: os.Args[1:]
main.go -> flag_parser.go: ParseOpsFlags(args, nil)
flag_parser.go -> pflag: fs.Parse(filtered)
flag_parser.go --> main.go: OpsFlags, positionals, err

alt ErrHelp
  main.go -> flag_parser.go: (usage already printed by fs.Usage closure)
  main.go -> opsfile_parser.go: ParseOpsFile (best-effort)
  main.go -> lister.go: FormatCommandList (best-effort)
  main.go -> os: Exit(0)
end

alt flags.Version
  main.go -> os: Printf version, Exit(0)
end

alt flags.List
  main.go -> opsfile_parser.go: ParseOpsFile
  main.go -> lister.go: FormatCommandList(os.Stdout, ...)
  main.go -> os: Exit(0)
end

main.go -> flag_parser.go: ParseOpsArgs(positionals)
main.go -> command_resolver.go: Resolve(cmd, env, commands, vars)

alt flags.DryRun
  main.go -> os: Println each resolved line (unless Silent)
else
  main.go -> executor.go: Execute(lines, shell)
end
```

### Key Design Decisions
- **pflag over stdlib `flag`:** pflag supports POSIX-style short flags (`-d`) and long flags (`--dry-run`) natively, which the standard `flag` package does not. This is the only external dependency in the project.
- **Help token stripping:** Help flags are manually stripped before `fs.Parse()` so that flags appearing after `--help` (e.g. `--help -D /path`) are still parsed. This ensures the `--directory` flag is available when rendering the best-effort command listing in help output.
- **`SetInterspersed(false)`:** Stops flag parsing at the first non-flag argument, matching stdlib `flag` behavior. This ensures positional arguments (env, command, command-args) are not misinterpreted as flags.
- **Best-effort help listing:** The `--help` path attempts to find and parse the Opsfile for the command listing but silently ignores errors, so help always works even without an Opsfile.

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/flag_parser.go` | Exists | Defines `OpsFlags`, `ErrHelp`, `ParseOpsFlags()`, `ParseOpsArgs()` |
| `internal/version.go` | Exists | Declares `Version` and `Commit` build-time variables |
| `internal/lister.go` | Exists | `FormatCommandList()` for `--list` and `--help` command listing |
| `cmd/ops/main.go` | Exists | Wires flag checks into the execution pipeline |
| `internal/flag_parser_test.go` | Exists | Tests for all flag parsing behavior |
| `internal/lister_test.go` | Exists | Tests for list output formatting |

---

## 5. Alternatives Considered

### Alternative A: Standard Library `flag` Package

**Description:** Use Go's built-in `flag` package instead of `pflag`.

**Pros:**
- Zero external dependencies
- Part of the standard library, always available

**Cons:**
- No POSIX short flag support (`-d` requires manual aliasing)
- No built-in `-?` support (not a valid flag name)
- More boilerplate for short/long flag pairs

**Why not chosen:** pflag provides a significantly better user experience with POSIX-style short flags and cleaner help output with minimal dependency cost. It is widely used in the Go ecosystem (used by cobra, kubectl, etc.).

### Alternative B: Cobra Command Framework

**Description:** Use the Cobra framework for full CLI command/subcommand support.

**Pros:**
- Rich subcommand support, auto-generated help, shell completion
- Very popular in Go ecosystem

**Cons:**
- Heavyweight dependency for a single-command CLI
- Adds complexity that `ops` doesn't need (no subcommands)
- Would change the invocation syntax

**Why not chosen:** `ops` is intentionally a flat CLI (`ops [flags] <env> <cmd>`) — Cobra's subcommand model doesn't fit. pflag (which Cobra uses internally) provides the needed flag features without the overhead.

---

## Open Questions
- [x] All current open questions resolved — feature is fully implemented

---

## 6. Task Breakdown

### Phase 1: Foundation (completed)
- [x] Define `OpsFlags` struct with all flag fields
- [x] Implement `ParseOpsFlags()` with pflag registration
- [x] Implement `ParseOpsArgs()` for positional argument extraction
- [x] Implement help token stripping for `-h`/`--help`/`-?`
- [x] Write unit tests for flag parsing

### Phase 2: Integration (completed)
- [x] Wire flag checks into `main.go` execution pipeline
- [x] Implement dry-run path in `main.go`
- [x] Implement version reporting with build-time ldflags
- [x] Implement `--list` flag with `FormatCommandList()`
- [x] Implement best-effort command listing in `--help` output
- [x] Write tests for lister formatting

### Phase 3: Polish (completed)
- [x] Ensure `--directory` interacts correctly with `--list` and `--help`
- [x] Update help/usage banner text
- [x] Update documentation

---
