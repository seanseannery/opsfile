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

---

## Shell Environment Variable Injection

### Overview

Shell environment variables (from the user's terminal session) can be referenced in Opsfile command lines using the same `$(VAR_NAME)` syntax already used for Opsfile-defined variables. This matches Makefile behaviour: no new syntax is introduced; environment variable lookup is added as a fallback in the existing resolution chain.

### Functional Requirements

- `$(VAR_NAME)` tokens that are valid identifiers are resolved using a four-level priority chain:
  1. Opsfile env-scoped variable (e.g. `prod_VAR_NAME`)
  2. Shell environment env-scoped variable (e.g. `os.Getenv("prod_VAR_NAME")`)
  3. Opsfile unscoped variable (e.g. `VAR_NAME`)
  4. Shell environment unscoped variable (e.g. `os.Getenv("VAR_NAME")`)
- If a variable is not found at any level, an error is returned (existing behaviour, unchanged)
- Shell environment variable lookup is case-sensitive; `$(PATH)` and `$(path)` are distinct
- Tokens that fail `isIdentifier()` (e.g. `$(aws ec2 describe-instances)`) continue to pass through unchanged — shell subcommand syntax is unaffected
- The feature is transparent: users do not need to declare which variables come from the environment vs. the Opsfile

### Syntax

Identical to existing Opsfile variable references:

```
# Opsfile

deploy:
    default:
        aws s3 cp ./dist s3://$(BUCKET)/$(APP_VERSION) --region $(AWS_REGION)
```

If `BUCKET` is defined in the Opsfile and `APP_VERSION` is not, `APP_VERSION` is resolved from the shell environment. `AWS_REGION` follows the same fallback chain.

### Priority Behaviour (Makefile-Compatible)

Makefiles treat environment variables as the lowest-priority source: a variable defined in the Makefile overrides an environment variable of the same name. `opsfile` mirrors this:

| Source | Priority |
|---|---|
| Opsfile env-scoped (`prod_VAR`) | 1 (highest) |
| Shell environment env-scoped (`$prod_VAR`) | 2 |
| Opsfile unscoped (`VAR`) | 3 |
| Shell environment unscoped (`$VAR`) | 4 (lowest) |

Env-scoped lookups (Opsfile then shell) are exhausted before falling back to unscoped lookups, so a `prod_`-prefixed shell variable takes precedence over an unscoped Opsfile definition.

### Implementation Overview

The only change required is in `internal/command_resolver.go` in `resolveVar`.

**Updated `resolveVar(varName, env string, vars OpsVariables) (string, error)`:**

1. Checks `vars[env+"_"+varName]` (env-scoped Opsfile variable) — returns value if found
2. Calls `os.Getenv(env+"_"+varName)` (env-scoped shell variable) — returns value if set
3. Checks `vars[varName]` (unscoped Opsfile variable) — returns value if found
4. Calls `os.Getenv(varName)` (unscoped shell variable) — returns value if set
5. Returns an error if all four lookups fail

No changes to `substituteVars`, `isIdentifier`, or any other part of the resolution pipeline.
