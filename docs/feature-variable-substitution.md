# Feature: Variable Substitution


## 1. Problem Statement & High-Level Goals

### Problem
Opsfile commands often need to reference configuration values that vary by environment (e.g. cluster names, regions, account IDs). Without variable substitution, users would have to duplicate commands across environments or rely entirely on shell environment variables with inconsistent syntax. Users also need the ability to use shell subcommands like `$(aws sts get-caller-identity)` without `ops` interfering with them.

### Goals
- [x] Replace `$(VAR_NAME)` tokens in command lines with values from Opsfile-defined variables
- [x] Support environment-scoped variable resolution (e.g. `prod_CLUSTER` takes precedence over `CLUSTER`)
- [x] Fall back to shell environment variables when a variable is not defined in the Opsfile
- [x] Pass through non-identifier `$(...)` tokens unchanged to preserve shell subcommand syntax

### Non-Goals
- No support for nested variable references (e.g. `$($(VAR))`)
- No expression evaluation inside `$(...)` — only bare identifiers are resolved
- No default/fallback values syntax (e.g. `$(VAR:-default)`)
- No variable escaping mechanism (e.g. `$$(VAR)` to produce a literal `$(VAR)`)

---

## 2. Functional Requirements

### FR-1: Variable Reference Syntax
`$(VAR_NAME)` tokens in shell lines are replaced with their resolved values. Only tokens whose content is a bare identifier (letters, digits, hyphens, underscores) are treated as variable references. Tokens containing spaces or non-identifier characters (e.g. `$(aws ec2 describe-instances)`) are passed through unchanged, preserving shell subcommand syntax.

### FR-2: Four-Level Priority Chain
Variable lookup uses a four-level priority chain, exhausting env-scoped lookups before falling back to unscoped:

1. **Opsfile env-scoped** — `vars["env_VAR_NAME"]` (e.g. `prod_CLUSTER`)
2. **Shell environment env-scoped** — `os.LookupEnv("env_VAR_NAME")` (e.g. `$prod_CLUSTER`)
3. **Opsfile unscoped** — `vars["VAR_NAME"]` (e.g. `CLUSTER`)
4. **Shell environment unscoped** — `os.LookupEnv("VAR_NAME")` (e.g. `$CLUSTER`)

This mirrors Makefile behavior: Opsfile-defined variables override shell environment variables at the same scope level.

### FR-3: Undefined Variable Error
If a variable reference cannot be resolved at any of the four priority levels, an error is returned. The error message includes the variable name and the environment being resolved.

### FR-4: Unclosed Token Handling
An unclosed `$(` (no matching `)`) is written through literally as `$(` — no error is raised. This allows partial shell syntax to pass through safely.

### FR-5: Identifier Detection
A token is considered an identifier if it is non-empty and consists only of letters (`a-z`, `A-Z`), digits (`0-9`), hyphens (`-`), and underscores (`_`). The `isIdentifier()` function in `opsfile_parser.go` implements this check and is shared with the parser.

### Example Usage

**Opsfile with variables and env-scoped overrides:**
```
CLUSTER=my-app
prod_CLUSTER=my-app-prod
AWS_REGION=us-east-1

tail-logs:
    default:
        aws logs tail /ecs/$(CLUSTER) --follow --region $(AWS_REGION)
    prod:
        aws logs tail /ecs/$(CLUSTER) --follow --region $(AWS_REGION)
```

Running `ops prod tail-logs` resolves to:
```bash
aws logs tail /ecs/my-app-prod --follow --region us-east-1
```
Here `$(CLUSTER)` resolves to `my-app-prod` (Opsfile env-scoped `prod_CLUSTER`) while `$(AWS_REGION)` resolves to `us-east-1` (Opsfile unscoped).

**Shell environment fallback:**
```
deploy:
    default:
        aws s3 cp ./dist s3://$(BUCKET)/$(APP_VERSION) --region $(AWS_REGION)
```
If `BUCKET` is defined in the Opsfile but `APP_VERSION` and `AWS_REGION` are not, they are resolved from the shell environment (`$APP_VERSION`, `$AWS_REGION`).

**Shell subcommand passthrough:**
```
show-caller:
    default:
        echo "Caller: $(aws sts get-caller-identity --query Account --output text)"
```
The `$(aws sts ...)` token contains spaces, so `isIdentifier()` returns false and the token passes through unchanged for the shell to evaluate.

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Performance | Variable substitution is O(n) per line where n is line length | Single-pass scan using `strings.Index` |
| NFR-2 | Compatibility | Shell environment lookup is case-sensitive | Matches POSIX behavior; `$(PATH)` and `$(path)` are distinct |
| NFR-3 | Reliability | Undefined variables produce clear error messages with variable name and environment | Prevents silent misconfiguration |
| NFR-4 | Compatibility | Shell subcommand syntax is never modified | `isIdentifier()` filter ensures only bare identifiers are resolved |
| NFR-5 | Maintainability | Substitution logic is isolated in `command_resolver.go` | Single file to modify for resolution behavior changes |

---

## 4. Architecture & Implementation Proposal

### Overview
Variable substitution is implemented in `internal/command_resolver.go` as part of the command resolution pipeline. After environment selection picks the correct shell lines for the requested environment, each line is scanned for `$(...)` tokens and resolved against the four-level variable priority chain. The `Resolve()` function orchestrates both environment selection and variable substitution.

### Component Design

**`Resolve(commandName, env, commands, vars)`** — Public entry point. Looks up the command by name, calls `selectLines()` for environment selection, then iterates each line through `substituteVars()`.

**`selectLines(cmd, env)`** — Selects shell lines for the given environment, falling back to the `default` block if the specific environment isn't defined. Returns an error if neither exists.

**`substituteVars(line, env, vars)`** — Scans a single line for `$(...)` tokens. For each token, checks `isIdentifier()` to determine if it's a variable reference or a shell subcommand. Variable references are resolved via `resolveVar()`; non-identifiers are passed through unchanged.

**`resolveVar(varName, env, vars)`** — Implements the four-level priority chain: Opsfile env-scoped → shell env-scoped → Opsfile unscoped → shell unscoped. Returns the first match or an error.

**`isIdentifier(s)`** (in `opsfile_parser.go`) — Returns true if the string is non-empty and contains only identifier characters (letters, digits, hyphens, underscores). Shared between the parser and resolver.

### Data Flow
```
Resolve(commandName, env, commands, vars)
  -> selectLines(cmd, env) -> []string (raw shell lines)
  -> for each line:
       substituteVars(line, env, vars)
         -> scan for "$(" markers
         -> extract token between "$(" and ")"
         -> isIdentifier(token)?
              yes -> resolveVar(token, env, vars) -> substituted value
              no  -> pass through "$(token)" unchanged
  -> ResolvedCommand{Lines: resolved}
```

#### Sequence Diagram
```
main.go -> command_resolver.go: Resolve(cmdName, env, commands, vars)
command_resolver.go -> command_resolver.go: selectLines(cmd, env)
  alt exact env match
    return lines for env
  else fallback
    return lines for "default"
  else no match
    return error
  end

loop each line
  command_resolver.go -> command_resolver.go: substituteVars(line, env, vars)
  loop each "$(" token
    command_resolver.go -> opsfile_parser.go: isIdentifier(token)
    alt is identifier
      command_resolver.go -> command_resolver.go: resolveVar(token, env, vars)
      alt Opsfile env-scoped found
        return value
      else shell env-scoped found
        return os.LookupEnv(env_VAR)
      else Opsfile unscoped found
        return value
      else shell unscoped found
        return os.LookupEnv(VAR)
      else not found
        return error
      end
    else not identifier
      pass through unchanged
    end
  end
end
command_resolver.go --> main.go: ResolvedCommand{Lines}
```

### Key Design Decisions
- **`strings.Index` scanning over regex:** A simple index-based scan is more readable, faster, and avoids regex compilation overhead. The `$(...)` syntax is straightforward enough that regex is unnecessary.
- **Shared `isIdentifier()` function:** Rather than duplicating identifier-checking logic, the resolver imports the same function used by the parser. This ensures consistent behavior for what constitutes a valid variable name.
- **Four-level priority chain:** Modeled after Makefile behavior where file-defined variables override environment variables. The env-scoped layer is exhausted before falling back to unscoped, so `prod_VAR` in the shell environment takes precedence over an unscoped `VAR` in the Opsfile.
- **`os.LookupEnv` over `os.Getenv`:** `LookupEnv` distinguishes between an unset variable and one set to an empty string, which is important for correct fallback behavior.
- **Error on undefined:** Rather than silently passing through undefined variables (which could lead to broken commands), an explicit error is returned. This catches typos and missing configuration early.

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/command_resolver.go` | Exists | `Resolve()`, `selectLines()`, `substituteVars()`, `resolveVar()` |
| `internal/opsfile_parser.go` | Exists | `isIdentifier()` and `isIdentChar()` shared helpers |
| `internal/command_resolver_test.go` | Exists | Tests for substitution, resolution priority, error cases |

---

## 5. Alternatives Considered

### Alternative A: Regex-Based Substitution

**Description:** Use `regexp.ReplaceAllStringFunc` with a pattern like `\$\(([^)]+)\)` to find and replace variable references.

**Pros:**
- More concise replacement logic
- Well-understood pattern matching

**Cons:**
- Regex compilation overhead on every line
- Harder to handle the identifier vs. non-identifier distinction cleanly
- Less readable for Go developers unfamiliar with the regex

**Why not chosen:** The index-based scan is simpler, faster, and makes the identifier check explicit. The `$(...)` syntax is simple enough that regex adds complexity without benefit.

### Alternative B: Go `text/template` Syntax

**Description:** Use Go's template syntax (`{{ .VAR }}`) instead of `$(VAR)`.

**Pros:**
- Built-in template engine with rich features
- No custom parser needed

**Cons:**
- Conflicts with shell syntax expectations — users expect `$(...)` from Makefile/shell familiarity
- Would require escaping `$()` in shell subcommands
- Template errors are harder to understand for ops users

**Why not chosen:** The `$(VAR)` syntax is familiar to Makefile and shell users, which is the target audience. Using Go templates would add cognitive overhead and break the Makefile-like mental model.

---

## Open Questions
- [x] All current open questions resolved — feature is fully implemented

---

## 6. Task Breakdown

### Phase 1: Foundation (completed)
- [x] Implement `substituteVars()` with `$(...)` token scanning
- [x] Implement `isIdentifier()` to distinguish variable refs from shell subcommands
- [x] Implement `resolveVar()` with Opsfile-only lookup (env-scoped then unscoped)
- [x] Write unit tests for substitution and identifier detection

### Phase 2: Shell Environment Integration (completed)
- [x] Extend `resolveVar()` with `os.LookupEnv` fallback at both scoped and unscoped levels
- [x] Implement four-level priority chain
- [x] Write tests for shell environment variable resolution and priority ordering

### Phase 3: Polish (completed)
- [x] Handle edge cases: unclosed `$(`, empty tokens, non-identifier tokens
- [x] Ensure error messages include variable name and environment context
- [x] Update documentation

---
