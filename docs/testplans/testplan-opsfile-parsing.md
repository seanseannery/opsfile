# Test Plan: Opsfile Parsing

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Parse variables from examples/Opsfile | Real examples/Opsfile | Variables `prod_AWS_ACCOUNT` and `preprod_AWS_ACCOUNT` extracted | unit |
| Parse commands from examples/Opsfile | Real examples/Opsfile | Commands `tail-logs` and `list-instance-ips` found | unit |
| Tail-logs environments parsed | Real examples/Opsfile | `default` and `local` environments with correct shell lines | unit |
| List-instance-ips environments parsed | Real examples/Opsfile | `prod` and `preprod` environments each with 1 joined line | unit |
| Comments skipped | Real examples/Opsfile | No keys or shell lines start with `#` | unit |
| Inline comment on unquoted variable | `PLAIN=123 # comment` | Value is `"123"` | unit |
| Double-quoted variable preserves `#` | `VAR="has#hash"` | Value is `"has#hash"` | unit |
| Single-quoted variable preserves `#` | `VAR='has#hash'` | Value is `"has#hash"` | unit |
| `#` without preceding space is part of value | `VAR=val#nospace` | Value is `"val#nospace"` | unit |
| Empty quoted string | `VAR=""` | Value is `""` | unit |
| Backslash continuation (single) | `cmd \` + next line | Lines joined into one | unit |
| Backslash continuation (chain) | Three lines joined by `\` | All joined into one line | unit |
| Backslash with space before `\` | `aws logs \` (space before backslash) | Trailing space stripped, lines joined | unit |
| Indent-based continuation | Deeper-indented line follows a shell line | Lines joined with space | unit |
| Indent continuation chain | Multiple deeper-indented lines | All joined to first line | unit |
| New command after indent continuation | Same-indent line after a continuation | Starts a new shell command | unit |
| Backslash at EOF | File ends with `\` on last line | Continuation buffer flushed, trailing `\` stripped | unit |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| File not found | Nonexistent path | Error (file open failure) | unit |
| Empty file | `""` | Error containing "empty" | unit |
| Only comments | `"# comment\n# comment\n"` | Error containing "empty" | unit |
| Duplicate command name | Two blocks with same command name | Error containing "duplicate" | unit |
| Variable with missing name | `=somevalue` | Error containing "missing name" | unit |
| Unclosed double quote | `VAR="unclosed` | Error containing "unclosed" | unit |
| Unclosed single quote | `VAR='unclosed` | Error containing "unclosed" | unit |

## Existing Automated Tests

### `internal/opsfile_parser_test.go`

- `TestParseOpsFile_Variables` (line 33) -- checks variables from examples/Opsfile
- `TestParseOpsFile_NoComments` (line 55) -- verifies comments don't leak
- `TestParseOpsFile_Commands` (line 77) -- checks command names from examples/Opsfile
- `TestParseOpsFile_TailLogsEnvironments` (line 100) -- checks `default` and `local` envs, backslash continuation
- `TestParseOpsFile_ListInstanceIpsEnvironments` (line 135) -- checks `prod`/`preprod` envs, backslash continuation
- `TestParseOpsFile_FileNotFound` (line 164) -- error for missing file
- `TestExtractVariableValue` (line 171) -- comprehensive unit test for value extraction (quotes, comments, edge cases)
- `TestParseOpsFile_InlineCommentOnVariable` (line 218) -- inline comment handling in full parse
- `TestParseOpsFile_UnclosedQuote` (line 246) -- unclosed quote error
- `TestParseOpsFile_EmptyFile` (line 263) -- empty file error
- `TestParseOpsFile_OnlyComments` (line 273) -- comment-only file error
- `TestParseOpsFile_DuplicateCommand` (line 284) -- duplicate command error
- `TestParseOpsFile_VariableMissingName` (line 303) -- `=value` with no name
- `TestParseOpsFile_BackslashContinuation` (line 320) -- single backslash join
- `TestParseOpsFile_BackslashContinuationChain` (line 341) -- chained backslash join
- `TestParseOpsFile_BackslashSpaceBeforeSlash` (line 363) -- space before `\`
- `TestParseOpsFile_IndentContinuation` (line 384) -- single indent join
- `TestParseOpsFile_IndentContinuationChain` (line 405) -- chained indent join
- `TestParseOpsFile_IndentNewCommandAfterContinuation` (line 427) -- new command at same indent
- `TestParseOpsFile_BackslashTrailingEOF` (line 451) -- `\` at end of file

## Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| Mixed tabs and spaces for indentation | unit | Consistent behavior or clear error for mixed indent |
| Very large Opsfile (many commands and variables) | unit | Parser handles files with 50+ commands without issues |
