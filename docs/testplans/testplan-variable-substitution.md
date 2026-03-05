# Test Plan: Variable Substitution

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Simple variable replacement | Line `echo $(VAR)`, `VAR=hello` | `echo hello` | unit |
| Scoped variable priority | `prod_VAR=scoped`, `VAR=unscoped`, env=`prod` | Scoped value used | unit |
| Unscoped fallback | Only `VAR=unscoped`, env=`prod` | Unscoped value used | unit |
| Non-identifier passthrough | `echo $(aws ec2 describe-instances)` | Passed through unchanged | unit |
| Unclosed `$(` written literally | `echo $(incomplete` | Output is `echo $(incomplete` | unit |
| Multiple variables in one line | `$(A) and $(B)` with both defined | Both substituted | unit |
| Variable adjacent to text | `prefix-$(VAR)-suffix` | `prefix-value-suffix` | unit |
| Variable with hyphen in name | `$(MY-VAR)` with `MY-VAR=val` | `val` substituted | unit |
| Variable with underscore in name | `$(MY_VAR)` with `MY_VAR=val` | `val` substituted | unit |
| Variable with digits in name | `$(VAR123)` with `VAR123=val` | `val` substituted | unit |
| Empty variable value | `$(VAR)` with `VAR=""` | Empty string substituted | unit |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Undefined variable | `echo $(MISSING)`, no matching variable | Error containing "not defined" | unit |
| Scoped and unscoped both missing | `$(VAR)` with neither `prod_VAR` nor `VAR` defined | Error containing "not defined" | unit |

## Existing Automated Tests

Variable substitution is tested indirectly through the `Resolve` function tests in `internal/command_resolver_test.go`:

- `TestResolve_DefaultFallback` (line 43) -- `$(AWS_ACCOUNT)` resolved via scoped `prod_AWS_ACCOUNT`
- `TestResolve_ScopedPriority` (line 84) -- scoped variable takes priority
- `TestResolve_UnscopedFallback` (line 102) -- unscoped fallback used
- `TestResolve_VariableNotDefined` (line 149) -- error for undefined variable
- `TestResolve_NonIdentifierPassthrough` (line 164) -- shell subcommand syntax preserved
- `TestResolve_MultiLineCommand` (line 180) -- multiple variables across multiple lines

## Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| Direct unit test for `substituteVars` | unit | Test `substituteVars` directly (not through `Resolve`) for isolated coverage — tested indirectly through `Resolve` |
| Nested `$(...)` syntax `$($(VAR))` | unit | Behavior with nested dollar-paren (likely treated as non-identifier inner token) |
| Variable value containing `$(...)` | unit | Substituted value itself contains `$(X)` -- whether recursive substitution occurs (it should not) |
