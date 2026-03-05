# Test Plan: Command Resolution

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Exact environment match | Command with `prod` and `preprod` envs, resolve for `prod` | Returns `prod` environment's shell lines | unit |
| Default fallback | Command with only `default` env, resolve for `prod` | Returns `default` environment's shell lines | unit |
| Local overrides default | Command with `default` and `local` envs, resolve for `local` | Returns `local` environment's lines, not `default` | unit |
| Multi-line command resolved | Command with 2 shell lines in environment | Both lines returned in order | unit |
| Variable substitution during resolution | Shell line with `$(VAR)`, variable defined | Variable replaced in output | unit |
| Scoped variable priority | Both `prod_VAR` and `VAR` defined | `prod_VAR` used when resolving for `prod` | unit |
| Unscoped variable fallback | Only `VAR` defined (no `prod_VAR`) | `VAR` value used when resolving for `prod` | unit |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Command not found | Resolve a command name not in the map | Error containing "not found" | unit |
| Environment not found, no default | Command has only `prod` env, resolve for `staging` | Error containing "no default" | unit |
| Undefined variable reference | Shell line references `$(MISSING_VAR)` with no matching variable | Error containing "not defined" | unit |

## Existing Automated Tests

### `internal/command_resolver_test.go`

- `TestResolve_ExactEnvMatch` (line 19) -- exact env match with multi-line output
- `TestResolve_DefaultFallback` (line 43) -- falls back to `default` with variable substitution
- `TestResolve_LocalOverridesDefault` (line 64) -- local env chosen over default
- `TestResolve_ScopedPriority` (line 84) -- `prod_VAR` takes priority over `VAR`
- `TestResolve_UnscopedFallback` (line 102) -- unscoped variable used when no scoped match
- `TestResolve_CommandNotFound` (line 119) -- error for nonexistent command
- `TestResolve_EnvNotFoundNoDefault` (line 134) -- error when no matching env and no default
- `TestResolve_VariableNotDefined` (line 149) -- error for undefined variable reference
- `TestResolve_NonIdentifierPassthrough` (line 164) -- `$(aws ec2 ...)` passed through unchanged
- `TestResolve_MultiLineCommand` (line 180) -- multi-line command with variable substitution

## Missing Automated Tests

All previously missing tests have been implemented and pass.
