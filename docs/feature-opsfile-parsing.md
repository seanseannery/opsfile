# Feature: Opsfile Parsing


## 1. Problem Statement & High-Level Goals

### Problem
The `ops` tool needs to read an Opsfile — a structured text file defining variables and commands — and produce typed data structures that downstream stages (command resolution, execution) can consume. The file format must be simple enough for users to author by hand, while supporting features like multi-environment commands, inline comments, and multi-line shell statements.

### Goals
- [x] Parse an Opsfile into variables (`OpsVariables`) and commands (`map[string]OpsCommand`)
- [x] Support per-environment shell blocks within each command
- [x] Support backslash and indent-based line continuation for multi-line shell commands
- [x] Handle quoted and unquoted variable values with inline comment stripping
- [x] Preserve command and environment declaration order for display purposes
- [x] Produce clear, line-numbered error messages for malformed files
- [x] Capture command descriptions from preceding `#` comment lines

### Non-Goals
- Variable resolution/substitution (handled by `command_resolver.go`)
- Executing shell lines (handled by `executor.go`)
- Supporting alternative file formats (YAML, TOML, JSON)

---

## 2. Functional Requirements

### FR-1: Variable Parsing
Variables are declared at the top level as `NAME=value` lines. Variable names are bare identifiers (letters, digits, hyphens, underscores). Values may be unquoted, double-quoted, or single-quoted. Quoted values preserve `#` characters inside; unquoted values treat `#` preceded by whitespace as an inline comment. Unclosed quotes and missing variable names are parse errors.

### FR-2: Command and Environment Block Parsing
Commands are declared as non-indented lines ending with `:` (e.g., `tail-logs:`). Duplicate command names are a parse error. Each command contains one or more environment blocks declared as indented lines ending with `:` (e.g., `  default:`, `  prod:`). Environment headers must be bare identifiers followed by `:`.

### FR-3: Shell Line Collection with Continuation
Shell lines are indented lines inside an environment block. Two continuation mechanisms are supported:
- **Backslash continuation**: a line ending with `\` is joined with the next line (the `\` is stripped)
- **Indent-based continuation**: a line indented deeper than the previous shell line is joined to it with a space

A new line at the same indent level as the previous shell line starts a new shell command.

### FR-4: Comments, Blank Lines, and Command Descriptions
Blank lines and lines starting with `#` (after trimming) are skipped during parsing. The first line of a comment block immediately preceding a command declaration is captured as that command's `Description` field. Blank lines reset the comment capture.

### FR-5: Validation and Error Reporting
An Opsfile containing no variables and no commands is rejected as empty. All parse error messages include the 1-based line number where the error occurred.

### FR-6: Ordered Output
The parser tracks and returns command declaration order and environment declaration order as separate `[]string` slices, used by the `--list` flag and help output for display.

### Example Usage

Given this Opsfile:
```
REGION=us-east-1
ACCOUNT=123456789  # AWS account

# Show recent application logs
tail-logs:
  default:
    echo "tailing logs for $REGION"
  prod:
    aws logs tail /prod/app \
      --follow \
      --region $(REGION)
```

The parser produces:
- **Variables**: `{"REGION": "us-east-1", "ACCOUNT": "123456789"}`
- **Commands**: `{"tail-logs": {Name: "tail-logs", Description: "Show recent application logs", Environments: {"default": [...], "prod": [...]}}}`
- **Command order**: `["tail-logs"]`
- **Env order**: `["default", "prod"]`

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Performance | Parse time should be negligible relative to command execution | Line-by-line streaming via `bufio.Scanner` |
| NFR-2 | Compatibility | Works on Linux, macOS, and Windows | No OS-specific file handling |
| NFR-3 | Reliability | All parse errors include line numbers and context | Uses `fmt.Errorf("line %d: %w", ...)` |
| NFR-4 | Security | No arbitrary code execution during parsing | Parser only reads and tokenizes text |
| NFR-5 | Maintainability | Comprehensive test suite covering edge cases | 30+ test cases in `opsfile_parser_test.go` |

---

## 4. Architecture & Implementation Proposal

### Overview
Parsing is implemented in `internal/opsfile_parser.go` using a line-by-line state machine. The `parser` struct holds mutable state as it scans through the file, dispatching each line to a handler based on the current parse state.

### Component Design

- **`ParseOpsFile(path)`** — Public entry point. Opens the file, creates a `parser`, scans lines via `bufio.Scanner`, calls `processLine()` for each, flushes any trailing continuation, validates, and returns results.
- **`parser` struct** — Internal state machine holding `variables`, `commands`, `state`, `currentCommand`, `currentEnv`, `continuationBuf`, `seenShellLine`, `lastShellIndent`, `lineNum`, `lastComment`, `order`, and `envOrder`.
- **`processLine(raw)`** — Dispatches each raw line based on `parseState`: `topLevel`, `inCommand`, or `inEnvironment`. Non-indented lines always reset state to `topLevel`.
- **`handleTopLevel(line)`** — Routes to `parseVariable()` (contains `=`) or `startCommand()` (ends with `:`).
- **`handleInCommand(line)`** — Detects environment headers via `isEnvHeader()`.
- **`handleInEnvironment(line, rawIndent)`** — Handles new env headers, backslash continuation, indent continuation, and regular shell lines.
- **`parseVariable(line)`** / **`extractVariableValue(raw)`** — Split on `=`, handle quoting and inline comment stripping.
- **`flushContinuation()`** — Appends any buffered backslash-continuation fragments as a complete shell line.
- **`joinLastShellLine(suffix)`** — Implements indent-based continuation by appending to the last collected shell line.
- **`validate()`** — Post-parse check that the file is not empty.
- **Helper functions**: `leadingWhitespace()`, `isEnvHeader()`, `isIdentifier()`, `isIdentChar()`, `indexComment()`.

### Data Flow
```
File (path) -> os.Open -> bufio.Scanner -> processLine() per line
                                              |
                          [topLevel] -> parseVariable() or startCommand()
                          [inCommand] -> handleInCommand() -> startEnv()
                          [inEnvironment] -> handleInEnvironment() -> appendShellLine()
                                              |
                          flushContinuation() -> validate() -> return (OpsVariables, map[string]OpsCommand, order, envOrder)
```

#### Sequence Diagram
```
User File               ParseOpsFile           parser state machine
   |                        |                        |
   |--- open file --------->|                        |
   |                        |--- scan lines -------->|
   |                        |                        |--- topLevel: variable or command?
   |                        |                        |--- inCommand: env header?
   |                        |                        |--- inEnvironment: shell line / continuation?
   |                        |<--- flush & validate --|
   |<--- return results ----|                        |
```

### Key Design Decisions
- **State machine over recursive descent:** The Opsfile format is simple and line-oriented, making a state machine clearer and simpler than a tree-based parser.
- **Streaming line-by-line:** Uses `bufio.Scanner` rather than reading the entire file into memory, keeping memory usage proportional to line length, not file size.
- **Two continuation mechanisms:** Backslash continuation (`\`) for explicit multi-line and indent-based continuation for natural readability — both common in shell-adjacent tools.
- **Comment-based descriptions:** Captures the comment line immediately above a command declaration as metadata, enabling `--list` to show human-readable command summaries without requiring a separate description syntax.
- **Order tracking:** Command and environment declaration order are tracked separately from the map keys, preserving the author's intended display order.

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/opsfile_parser.go` | Existing | Core parser: `ParseOpsFile()`, state machine, variable/command/env parsing, continuation logic |
| `internal/opsfile_parser_test.go` | Existing | 30+ test cases covering variables, commands, environments, continuations, edge cases, errors, ordering, and descriptions |

---

## 5. Alternatives Considered

### Alternative A: Regex-Based Parsing

**Description:** Use regular expressions to match variable declarations, command headers, environment headers, and shell lines.

**Pros:**
- Concise pattern matching for simple cases
- Familiar to many developers

**Cons:**
- Difficult to handle stateful constructs like continuation lines and nested blocks
- Harder to produce line-numbered error messages
- Complex regexes become unreadable quickly

**Why not chosen:** The state machine approach is more maintainable for a format with contextual meaning (indentation, continuation) and produces better error messages.

---

### Alternative B: Use an Existing Config Parser (YAML/TOML Library)

**Description:** Define the Opsfile format as YAML or TOML and use an existing parsing library.

**Pros:**
- Mature, well-tested parsing
- Standardized format familiar to users

**Cons:**
- Adds an external dependency
- Neither YAML nor TOML naturally maps to the Opsfile's command/environment/shell-lines structure
- Less control over error messages and file format evolution
- YAML indentation semantics differ from the Opsfile convention

**Why not chosen:** A custom format allows the simplest possible syntax for the use case (operations commands) without forcing users to learn YAML/TOML rules. Zero external dependencies is a project goal.

---

## Open Questions
- (none — feature is fully implemented)

---

## 6. Task Breakdown

> *Retrospective — all tasks completed in initial implementation.*

### Phase 1: Foundation
- [x] Define `OpsVariables`, `OpsCommand`, `parseState`, and `parser` types
- [x] Implement `ParseOpsFile()` entry point with file open and line scanning
- [x] Implement `processLine()` state dispatch and `handleTopLevel()`
- [x] Implement `parseVariable()` and `extractVariableValue()` with quote/comment handling

### Phase 2: Command and Environment Parsing
- [x] Implement `startCommand()` with duplicate detection
- [x] Implement `handleInCommand()` and `startEnv()` for environment blocks
- [x] Implement `handleInEnvironment()` with shell line collection
- [x] Implement backslash continuation (`flushContinuation()`)
- [x] Implement indent-based continuation (`joinLastShellLine()`)

### Phase 3: Polish
- [x] Add `validate()` for empty-file rejection
- [x] Add line-number context to error messages
- [x] Add command description capture from preceding comment lines
- [x] Track and return command order and environment order
- [x] Write comprehensive unit tests (30+ cases)

---
