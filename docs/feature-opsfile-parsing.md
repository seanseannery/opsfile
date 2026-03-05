# Opsfile Parsing

## Functional Requirements

- The parser reads an Opsfile and produces two outputs: a map of variables and a map of commands
- **Variables** are declared at the top level as `NAME=value` lines
  - Variable names are bare identifiers (letters, digits, hyphens, underscores)
  - Values may be unquoted, double-quoted, or single-quoted
  - Quoted values preserve `#` characters inside; unquoted values treat `#` preceded by whitespace as an inline comment
  - Unclosed quotes are a parse error
  - A `=` with no name on the left is a parse error
- **Commands** are declared as top-level lines ending with `:` (e.g. `tail-logs:`)
  - Duplicate command names are a parse error
  - Each command contains one or more **environment blocks**, declared as indented lines ending with `:` (e.g. `  default:`, `  prod:`)
  - Environment headers must be bare identifiers followed by `:`
- **Shell lines** are indented lines inside an environment block
  - **Backslash continuation**: a line ending with `\` is joined with the next line (the `\` is stripped)
  - **Indent-based continuation**: a line indented deeper than the previous shell line is joined to it with a space
  - A new line at the same indent level as the previous shell line starts a new shell command
- Blank lines and lines starting with `#` (after trimming) are skipped
- An Opsfile containing no variables and no commands is rejected as empty
- Line numbers are included in parse error messages

## Implementation Overview

Parsing is implemented in `internal/opsfile_parser.go` using a line-by-line state machine.

**State machine:**

The `parser` struct tracks state via a `parseState` enum with three values:

- `topLevel` -- expecting variables (`NAME=value`) or command headers (`name:`)
- `inCommand` -- inside a command block, expecting environment headers (`env:`)
- `inEnvironment` -- inside an environment block, collecting shell lines

A non-indented line always resets state to `topLevel`.

**Key functions:**

- `ParseOpsFile(path string) (OpsVariables, map[string]OpsCommand, error)` -- public entry point; opens the file, creates a `parser`, scans lines, validates, and returns results
- `processLine(raw string)` -- dispatches each raw line to the correct handler based on current state
- `handleTopLevel(line string)` -- routes to `parseVariable` or `startCommand`
- `handleInCommand(line string)` -- detects environment headers via `isEnvHeader`
- `handleInEnvironment(line string, rawIndent int)` -- handles shell lines, backslash continuation, indent continuation, and new environment headers
- `parseVariable(line string)` -- splits on `=`, calls `extractVariableValue` for comment/quote handling
- `extractVariableValue(raw string)` -- strips quotes or inline comments from the value portion
- `flushContinuation()` -- appends any buffered backslash-continuation fragments as a complete shell line
- `joinLastShellLine(suffix string)` -- implements indent-based continuation by appending to the last collected shell line
- `validate()` -- post-parse check that the file is not empty

**Key types:**

- `OpsVariables` -- `map[string]string` of variable name to value
- `OpsCommand` -- struct with `Name string` and `Environments map[string][]string`
- `parser` -- internal struct holding `variables`, `commands`, `state`, `currentCommand`, `currentEnv`, `continuationBuf`, `lastShellIndent`, `lineNum`
