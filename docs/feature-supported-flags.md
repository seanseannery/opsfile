# Supported CLI Flags

## Dry-Run Mode

### Functional Requirements

- When `--dry-run` or `-d` is passed, `ops` prints the fully resolved shell lines to stdout without executing them
- Each line is printed on its own line via `fmt.Println`
- If `--silent` / `-s` is also set, even the dry-run output is suppressed (nothing is printed and nothing is executed)
- Variable substitution and environment selection still occur in dry-run mode -- the user sees the exact commands that would be executed

### Implementation Overview

Dry-run mode is handled in `cmd/ops/main.go` in the `main()` function.

**Data flow:**

1. After `Resolve()` produces a `ResolvedCommand`, `main()` checks `flags.DryRun`
2. If true and `!flags.Silent`, it loops over `resolved.Lines` and prints each with `fmt.Println`
3. The function returns immediately without calling `Execute()`

**Key symbols:**

- `OpsFlags.DryRun` -- the flag value from `ParseOpsFlags()`
- `OpsFlags.Silent` -- checked in combination with `DryRun` to suppress output
- The dry-run branch in `main()` at approximately lines 63-69

## Help / Usage Output

### Functional Requirements

- When `-h`, `--help`, or `-?` is passed, `ops` prints a usage banner and flag descriptions to stderr, then exits with code 0
- The usage banner includes:
  - A description of what `ops` does
  - The invocation syntax: `ops [flags] <environment> <command> [command-args]`
  - Examples: `ops preprod open-dashboard`, `ops --dry-run prod tail-logs`
  - A list of all available flags with their descriptions
- `-?` is handled specially before `flag.Parse` because it is not a valid Go `flag` package flag name

### Implementation Overview

Help output is implemented in `internal/flag_parser.go` within `ParseOpsFlags()`.

**Data flow:**

1. Before `fs.Parse()`, `slices.Contains(osArgs, "-?")` checks for the `-?` token; if found, calls `fs.Usage()` and returns `ErrHelp`
2. For `-h` and `--help`, the standard `flag` package triggers `fs.Usage()` and returns `flag.ErrHelp`, which is translated to the `ErrHelp` sentinel
3. `fs.Usage` is a closure that writes the banner text with `fmt.Fprint(fs.Output(), ...)` and then calls `fs.PrintDefaults()` for flag descriptions
4. In `main()`, `errors.Is(err, internal.ErrHelp)` catches the sentinel and calls `os.Exit(0)`

**Key symbols:**

- `ErrHelp` -- package-level sentinel: `errors.New("help requested")`
- `fs.Usage` closure in `ParseOpsFlags()` -- defines the help text
- The `-?` pre-check using `slices.Contains`

## Version Reporting

### Functional Requirements

- When `-v` or `--version` is passed, `ops` prints its version string and exits with code 0
- The output format is: `ops version <version> (<os>/<arch>)` (e.g. `ops version 0.0.1 (darwin/arm64)`)
- The version is set at build time via Go linker flags; the default is `0.0.1`

### Implementation Overview

Version reporting is split across two files.

**`internal/version.go`:**

- Declares `var Version = "0.0.1"` -- the default version
- Build-time override: `go build -ldflags="-X sean_seannery/opsfile/internal.Version=1.2.3" ./cmd/ops/`

**`cmd/ops/main.go`:**

- After parsing flags, checks `flags.Version`
- If true, prints `fmt.Printf("ops version %s (%s/%s)\n", internal.Version, runtime.GOOS, runtime.GOARCH)` and calls `os.Exit(0)`

## List Commands (`--list` / `-l`) — Issue #19

### Problem

There is no way to discover what commands and environments an Opsfile provides without opening the file and reading it manually. Under time-pressure (e.g. an incident), this adds unnecessary friction.

### Functional Requirements

- When `--list` or `-l` is passed, `ops` prints a summary of the Opsfile's available environments and commands, then exits with code 0
- **Environment listing:** all unique, explicitly defined environment names across every command are collected into a sorted, deduplicated list and printed under an "Environments" heading. The synthetic name `default` is included only if at least one command defines a `default:` block
- **Command listing:** every command is printed in Opsfile-declaration order with its description (if any) and the environments it supports
- **Descriptions from comments:** a `#` comment line immediately preceding a command declaration (e.g. `# Tail CloudWatch logs`) is captured as that command's description. Only the single comment line directly above the command name line is used. If no such comment exists, the description is left blank (no error)
- **No execution:** when `--list` is active, no command resolution or execution occurs — even if an environment and command are supplied on the CLI
- **`--help` integration:** when `-h`, `--help`, or `-?` is passed, after printing the standard flag/usage text, `ops` attempts to locate and parse the Opsfile and appends the same command listing. If the Opsfile cannot be found or parsed, the help output is still shown (silent failure for the listing portion only)
- **`--directory` interaction:** if `-D <dir>` is combined with `--list`, the listing uses the Opsfile from the specified directory

### Example Output

Given the example Opsfile at `examples/Opsfile`:

```
$ ops --list

Commands Found in [./relative/path/to/Opsfile]:

Environments:
  default  local  preprod  prod

Commands:
  show-profile       Using AWS profile
  tail-logs          Tail CloudWatch logs
  list-instance-ips  List the private IPs of running instances
```

When no descriptions exist, the right-hand column is simply empty:

```
Commands:
  show-profile
  tail-logs
  list-instance-ips
```

### Acceptance Criteria

1. `-l` and `--list` both trigger the listing and exit 0
2. Environments are deduplicated across all commands and listed in the order they appear in Opsfile
3. Commands are listed in the order they appear in the Opsfile (parser insertion order)
4. Description is extracted from the first line of a `# comment` block immediately above the `command-name:` declaration; for multi-line comment blocks, the first line (title/summary) is used as the description
5. Missing description produces a blank — no error
6. Passing `--list` alongside a command and environment does not execute anything
7. `--help` appends the command listing after the flag summary; Opsfile errors are silently ignored in this path
8. Unit tests for description extraction in the parser
9. Integration tests for `--list` output format (environments + commands + descriptions)
10. Integration test confirming `--help` shows command listing when an Opsfile is available
11. README.md, this doc, github site and the help/usage banner updated with the new flag
12. Example Opsfiles do not need changes — existing `#` comments already follow the convention

### Implementation Plan

The feature touches four areas: the parser (description capture), the flag parser (new flag), a new lister module (formatting), and `main.go` (wiring).

#### 1. Parser — capture command descriptions (`internal/opsfile_parser.go`)

- Add a `Description` field to `OpsCommand`:
  ```go
  type OpsCommand struct {
      Name         string
      Description  string                 // from # comment above declaration
      Environments map[string][]string
  }
  ```
- Add a `lastComment string` field to the `parser` struct. On each call to `processLine`:
  - If the trimmed line starts with `#`, store the text (minus the `# ` prefix) in `lastComment`
  - If the trimmed line is blank, clear `lastComment` (a blank line between comment and command breaks the association)
  - When `startCommand()` is called, copy `lastComment` into the new command's `Description` and clear `lastComment`
- This is the only parser change. Comments are currently discarded at line 89 of `processLine` — the `#`-prefix check needs to happen before the early return, capturing the text before discarding the line from command parsing

#### 2. Flag parser — add `--list` / `-l` (`internal/flag_parser.go`)

- Add `List bool` to `OpsFlags`
- Register with pflag: `fs.BoolP("list", "l", false, "list available commands and environments")`
- Wire `*list` into the returned `OpsFlags`
- Update `fs.Usage` closure to include the new flag in the usage banner (pflag handles this automatically via `PrintDefaults`)

#### 3. New lister module (`internal/lister.go`)

Create a new file with a single public function:

```go
// FormatCommandList writes a human-readable summary of environments and
// commands to w. Commands are printed in the order provided by cmds.
func FormatCommandList(w io.Writer, cmds []OpsCommand)
```

- **Why a slice, not a map?** Maps don't preserve insertion order. The parser should return an ordered slice (or `main.go` can build one from the map using the order tracked during parsing — see note below).
- **Environment collection:** iterate all commands, collect environment names into a `map[string]struct{}` set, sort the keys, and print them space-separated under an `Environments:` header
- **Command table:** compute the max command-name length for column alignment, then print each command name left-padded with two spaces followed by the description (if any)
- The function writes to an `io.Writer` so tests can capture output without touching stdout

**Parser ordering note:** the current parser returns `map[string]OpsCommand`. To preserve declaration order for the listing, add an `order []string` slice to the `parser` struct, appending each command name in `startCommand()`. Expose this as a second return value from `ParseOpsFile` or embed order in a wrapper type. The simplest approach: return an additional `[]string` (command names in order) from `ParseOpsFile`:

```go
func ParseOpsFile(path string) (OpsVariables, map[string]OpsCommand, []string, error)
```

All existing callers pass `_` for the new return value, so this is backward-compatible at the call site.

#### 4. Main wiring (`cmd/ops/main.go`)

**`--list` path:**

```
flags parsed → find Opsfile → ParseOpsFile → if flags.List → FormatCommandList(os.Stdout, ...) → os.Exit(0)
```

Insert this check after `ParseOpsFile` succeeds and before `ParseOpsArgs`. This means `--list` does not require positional args.

**`--help` path:**

After the existing `errors.Is(err, internal.ErrHelp)` block (which currently calls `os.Exit(0)`), insert a best-effort listing:

```go
if errors.Is(err, internal.ErrHelp) {
    // Best-effort: try to show available commands
    if dir, derr := resolveOpsfileDir(flags.Directory); derr == nil {
        if _, cmds, order, perr := internal.ParseOpsFile(filepath.Join(dir, opsFileName)); perr == nil {
            fmt.Fprintln(os.Stderr)
            internal.FormatCommandList(os.Stderr, /* ordered cmds */)
        }
    }
    os.Exit(0)
}
```

Note: the help listing writes to stderr (matching the existing help output destination).

#### 5. Tests

| Test file | What it covers |
|-----------|---------------|
| `internal/opsfile_parser_test.go` | Description extraction: comment directly above command, blank line gap clears description, no comment yields empty string, multi-line comments only use last line |
| `internal/lister_test.go` | `FormatCommandList` output format: column alignment, sorted environments, empty descriptions, single command, many commands |
| `internal/flag_parser_test.go` | `--list` and `-l` parsed correctly into `OpsFlags.List` |
| `cmd/ops/main_test.go` or integration test | End-to-end: `ops --list` against example Opsfile produces expected output; `ops --help` includes command listing; `ops --list prod tail-logs` does not execute |

#### 6. Documentation updates

- `docs/feature-supported-flags.md` — this section (already done)
- `README.md` — add `--list` to the flags table and add a usage example
- Help/usage banner in `flag_parser.go` — pflag auto-includes registered flags in `PrintDefaults()`, so no manual text change needed beyond the registration

### Files Changed (summary)

| File | Change |
|------|--------|
| `internal/opsfile_parser.go` | Add `Description` field to `OpsCommand`, capture `lastComment` in parser, return command order |
| `internal/flag_parser.go` | Add `List` to `OpsFlags`, register `--list`/`-l` flag |
| `internal/lister.go` | **New file** — `FormatCommandList()` |
| `cmd/ops/main.go` | Wire `--list` exit path, add command listing to `--help` path |
| `internal/opsfile_parser_test.go` | Tests for description capture and ordering |
| `internal/lister_test.go` | **New file** — tests for output formatting |
| `internal/flag_parser_test.go` | Tests for new flag |
| `README.md` | Document `--list` flag |
| `docs/feature-supported-flags.md` | This section |
