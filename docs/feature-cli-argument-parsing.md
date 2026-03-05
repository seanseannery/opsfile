
# CLI Arguments and Flags Parsing

The `ops` cli takes input from the user when invoked through flags (which modify how the tool runs) and arguments (which select what commands to invoke as defined in the `Opsfile`).  This doc describes the requirements and implementation overview.

## CLI Flag Parsing

### Functional Requirements

- The CLI accepts the following flags before positional arguments:
  - `-D <directory>` / `--directory <directory>` -- use the Opsfile in the given directory instead of discovering it
  - `-d` / `--dry-run` -- print resolved commands without executing them
  - `-s` / `--silent` -- suppress output during execution (also suppresses dry-run output)
  - `-v` / `--version` -- print the ops version string and exit
  - `-h` / `--help` / `-?` -- print usage text and exit
- Unknown flags produce an error
- Flags must appear before positional arguments; once positional arguments begin, remaining tokens are treated as positionals

### Implementation Overview

Flag parsing is implemented in `internal/flag_parser.go` in the function `ParseOpsFlags()`.

**Data flow:**

1. A `flag.FlagSet` named `"ops"` is created with `ContinueOnError` so the caller controls error handling
2. Each flag is registered twice (short and long form) using `fs.String`/`fs.Bool` and their `Var` counterparts sharing the same pointer
3. Before calling `fs.Parse()`, the `-?` token is checked manually via `slices.Contains` because `-?` is not a valid `flag` package flag name
4. `fs.Parse(osArgs)` processes the flags; `flag.ErrHelp` is translated to the package-level sentinel `ErrHelp`
5. The function returns an `OpsFlags` struct and the remaining positional arguments via `fs.Args()`

**Key types and symbols:**

- `OpsFlags` struct -- holds `Directory string`, `DryRun bool`, `Silent bool`, `Version bool`
- `ErrHelp` sentinel -- returned when help is requested; checked with `errors.Is` in `main()`
- `ParseOpsFlags(osArgs []string) (OpsFlags, []string, error)` -- the public API
- `fs.Usage` closure -- prints the usage banner and flag defaults to stderr

## CLI (Non-Flag) Argument Parsing

### Functional Requirements

- After flags are consumed, the remaining positional arguments follow the structure: `<environment> <command> [command-args...]`
- The first positional argument is the environment name (e.g. `prod`, `preprod`, `local`)
- The second positional argument is the command name (e.g. `tail-logs`, `list-instance-ips`)
- Any additional positional arguments are passthrough command args, preserved in order
- Missing environment produces an error: `"missing environment argument"`
- Missing command (only one positional) produces an error: `"missing command argument"`

### Implementation Overview

Argument parsing is implemented in `internal/flag_parser.go` in the function `ParseOpsArgs()`.

**Data flow:**

1. `ParseOpsArgs` receives the `[]string` slice returned as the second value from `ParseOpsFlags`
2. It checks `len(nonFlagArgs)` -- error if less than 1 (no env) or less than 2 (no command)
3. Returns an `Args` struct with fields populated from slice positions:
   - `OpsEnv = nonFlagArgs[0]`
   - `OpsCommand = nonFlagArgs[1]`
   - `CommandArgs = nonFlagArgs[2:]`

**Key types and symbols:**

- `Args` struct -- holds `OpsEnv string`, `OpsCommand string`, `CommandArgs []string`
- `ParseOpsArgs(nonFlagArgs []string) (Args, error)` -- the public API
- Note: `CommandArgs` is currently parsed but not yet plumbed through to command execution (the field exists for future use)


