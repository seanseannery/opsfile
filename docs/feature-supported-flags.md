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
