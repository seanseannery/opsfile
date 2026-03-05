# Test Plan: Shell Environment Variable Injection

## Green Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Unscoped shell env fallback | `$(VAR)`, no Opsfile vars, `VAR=hello` in shell env, env=`prod` | `hello` substituted | unit |
| Env-scoped shell env used | `$(VAR)`, no Opsfile vars, `prod_VAR=scoped` in shell env, env=`prod` | `scoped` substituted | unit |
| Opsfile env-scoped beats shell env-scoped | `$(VAR)`, `prod_VAR=opsfile`, `prod_VAR=shell` in shell env, env=`prod` | Opsfile value `opsfile` used | unit |
| Shell env-scoped beats Opsfile unscoped | `$(VAR)`, `VAR=opsfile-unscoped`, `prod_VAR=shell-scoped` in shell env, env=`prod` | Shell scoped value `shell-scoped` used | unit |
| Opsfile unscoped beats shell env unscoped | `$(VAR)`, `VAR=opsfile-unscoped`, `VAR=shell-unscoped` in shell env, env=`prod` | Opsfile value `opsfile-unscoped` used | unit |
| Shell env unscoped is last resort | `$(VAR)`, no Opsfile vars, no scoped shell env var, `VAR=shell` in shell env, env=`prod` | `shell` substituted | unit |
| Full priority chain — level 1 wins | All four sources set for same name, env=`prod` | Opsfile `prod_VAR` value used | unit |
| Full priority chain — level 2 wins | No `prod_VAR` in Opsfile; `prod_VAR` set in shell env, `VAR` in Opsfile and shell | Shell `prod_VAR` value used | unit |
| Full priority chain — level 3 wins | No scoped sources; `VAR` in Opsfile and shell env | Opsfile `VAR` value used | unit |
| Full priority chain — level 4 wins | No Opsfile vars; no scoped shell env; `VAR` in shell env | Shell `VAR` value used | unit |
| Multiple variables mixed sources | Line uses `$(A)` (Opsfile) and `$(B)` (shell env) | Both substituted from respective sources | unit |
| Shell env var with empty string value | `$(VAR)`, `VAR=""` in shell env, no Opsfile definition | Empty string substituted (not an error) | unit |
| Non-identifier token unaffected | `$(aws ec2 describe-instances)`, `aws` set in shell env | Token passed through unchanged | unit |

## Red Path Tests

| Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|
| Variable absent from all four sources | `$(MISSING)`, not in Opsfile, not in shell env | Error returned | unit |
| Scoped shell env absent, all others absent | `$(VAR)`, `prod_VAR` not in Opsfile or shell env, `VAR` not defined anywhere | Error returned | unit |

## Existing Automated Tests

None — shell environment variable injection is a new feature not yet covered by any tests in `internal/command_resolver_test.go`.

## Missing Automated Tests

All tests in this plan are missing. Recommended test locations and approaches:

| Scenario | Type | Location | Notes |
|---|---|---|---|
| Unscoped shell env fallback | unit | `internal/command_resolver_test.go` | Use `t.Setenv` to inject shell env var; no Opsfile vars defined |
| Env-scoped shell env used | unit | `internal/command_resolver_test.go` | `t.Setenv("prod_VAR", ...)` |
| Opsfile env-scoped beats shell env-scoped | unit | `internal/command_resolver_test.go` | Both set; assert Opsfile value wins |
| Shell env-scoped beats Opsfile unscoped | unit | `internal/command_resolver_test.go` | `prod_VAR` in shell, `VAR` in Opsfile; assert shell scoped wins |
| Opsfile unscoped beats shell env unscoped | unit | `internal/command_resolver_test.go` | `VAR` in both; assert Opsfile wins |
| Shell env unscoped is last resort | unit | `internal/command_resolver_test.go` | Only `VAR` in shell env |
| Full four-level priority table | unit | `internal/command_resolver_test.go` | Table-driven test covering all four winner scenarios |
| Mixed sources across multiple variables | unit | `internal/command_resolver_test.go` | Two `$(X)` tokens, each resolved from a different source |
| Empty shell env value substituted | unit | `internal/command_resolver_test.go` | `t.Setenv("VAR", "")` — assert empty string, not error |
| Non-identifier unaffected by shell env | unit | `internal/command_resolver_test.go` | Confirm passthrough is unaffected even when shell env matches partial token |
| Variable absent from all sources | unit | `internal/command_resolver_test.go` | Confirm error when shell env also has no match |
