# Command Execution

When the use passes in arguments to the `ops` cli.  It must first map the user's input to the commands as defined in the `Opsfile`. If it 
cannot find the command, it will fail. Once, successfully mapped it must execute the resolved commands in the shell for the user, substituting 
any variables values defined in the `Opsfile` into the command.

## Command Resolution

### Functional Requirements

- Given a command name and environment, the resolver selects the correct set of shell lines to execute
- Environment selection follows this priority:
  1. Exact match on the requested environment (e.g. `prod`)
  2. Fallback to the `default` environment block if the specific one is absent
  3. Error if neither the specific environment nor `default` exists
- If the command name does not exist in the parsed commands map, an error is returned
- After environment selection, all `$(VAR_NAME)` tokens in the shell lines are substituted with variable values (see Variable Substitution feature)

### Implementation Overview

Command resolution is implemented in `internal/command_resolver.go`.

**Data flow:**

1. `Resolve(commandName, env string, commands map[string]OpsCommand, vars OpsVariables) (ResolvedCommand, error)` is the public entry point
2. It looks up the command by name in the `commands` map -- error if not found
3. Calls `selectLines(cmd OpsCommand, env string) ([]string, error)` which checks `cmd.Environments[env]` first, then `cmd.Environments["default"]`
4. Iterates over the selected shell lines and calls `substituteVars` on each one
5. Returns a `ResolvedCommand{Lines: []string}` containing the fully substituted shell lines

**Key types and symbols:**

- `ResolvedCommand` struct -- holds `Lines []string`, the final shell lines ready for execution
- `Resolve()` -- orchestrates lookup, environment selection, and variable substitution
- `selectLines()` -- implements the env-then-default fallback logic

## Shell Execution

### Functional Requirements

- Resolved shell lines are executed sequentially, one at a time
- Each line is run via the user's `$SHELL` environment variable; if unset, `/bin/sh` is used as the fallback
- Each command is invoked as `<shell> -c <line>`
- Commands inherit the current process environment, stdin, stdout, and stderr (fully interactive)
- Execution stops immediately on the first command that returns a non-zero exit code
- The exit code from a failed command is propagated as the exit code of the `ops` process itself
- If no shell lines are provided (empty list), execution is a no-op

### Implementation Overview

Shell execution is implemented in `internal/executor.go`.

**`Execute(lines []string, shell string) error`:**

1. Iterates over `lines` sequentially
2. For each line, creates `exec.Command(shell, "-c", line)`
3. Wires `cmd.Stdin`, `cmd.Stdout`, `cmd.Stderr` to `os.Stdin`, `os.Stdout`, `os.Stderr`
4. Calls `cmd.Run()` -- blocks until the command completes
5. On error, returns `fmt.Errorf("running %q: %w", line, err)` which wraps the underlying `*exec.ExitError`
6. In `main()`, the error is unwrapped with `errors.As(err, &exitErr)` to extract and propagate the exit code via `os.Exit`

**Key symbols:**

- `Execute(lines []string, shell string) error` -- the public API
- Shell selection logic in `cmd/ops/main.go`: `os.Getenv("SHELL")` with `/bin/sh` fallback
