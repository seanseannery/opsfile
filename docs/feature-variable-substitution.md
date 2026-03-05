# Variable Substitution

## Functional Requirements

- `$(VAR_NAME)` tokens in shell lines are replaced with their values from the Opsfile variables
- Only tokens whose content is a bare identifier (letters, digits, hyphens, underscores) are treated as Opsfile variable references
- Tokens containing spaces or non-identifier characters (e.g. `$(aws ec2 describe-instances)`) are passed through unchanged, preserving shell subcommand syntax
- Variable lookup uses environment-scoped priority:
  1. `env_VAR_NAME` is checked first (e.g. for env `prod` and variable `CLUSTER`, check `prod_CLUSTER`)
  2. `VAR_NAME` is checked as a fallback (unscoped)
  3. If neither is defined, an error is returned
- An unclosed `$(` (no matching `)`) is written through literally as `$(`

## Implementation Overview

Variable substitution is implemented in `internal/command_resolver.go` in two functions.

**`substituteVars(line, env string, vars OpsVariables) (string, error)`:**

1. Scans `line` for `$(` markers using `strings.Index`
2. For each `$(...)` token, extracts the content between `$(` and `)`
3. Calls `isIdentifier(token)` to decide whether this is a variable reference or a shell subcommand
4. If identifier: calls `resolveVar` to look up the value and writes it to a `strings.Builder`
5. If not identifier: writes `$(`, the token, and `)` back unchanged
6. If no closing `)` is found, writes `$(` literally and continues scanning

**`resolveVar(varName, env string, vars OpsVariables) (string, error)`:**

1. Checks `vars[env+"_"+varName]` (scoped lookup)
2. Falls back to `vars[varName]` (unscoped lookup)
3. Returns an error if neither key exists

**Key symbols:**

- `substituteVars()` -- per-line substitution engine
- `resolveVar()` -- env-scoped variable lookup with fallback
- `isIdentifier()` (from `opsfile_parser.go`) -- shared helper that determines whether a `$(...)` token is a variable reference
