# Test Plan: [Feature Name]

**Feature Doc:** [Link to feature document if applicable]

---

## 1. Scope

### In Scope
- [Component or behavior being tested]
- [Component or behavior being tested]
- [Regression Testing for dependant feature]

### Out of Scope
- [What is explicitly NOT covered by this test plan]

---

## 2. Test Objectives

- [What this test plan aims to validate — e.g., "Verify flag parsing handles all documented short/long forms"]
- [Secondary objective if applicable]

---

## 3. Entry & Exit Criteria

### Entry Criteria (prerequisites before testing begins)
- [ ] Feature implementation is code-complete
- [ ] Code compiles without errors (`make build`)
- [ ] Existing tests still pass (`make test`)

### Exit Criteria (conditions to consider testing complete)
- [ ] All green path automated tests pass
- [ ] All red path automated tests pass
- [ ] All green/red manual tests have been run and pass
- [ ] No regressions in related features
- [ ] Test coverage does not decrease

### Acceptance Criteria

- [ ] [Criterion that must be true for the feature to be considered working — e.g., "All documented flag forms are parsed correctly"]
- [ ] [Criterion — e.g., "Error messages include actionable context"]
- [ ] [Criterion — e.g., "No breaking changes to existing Opsfile behavior"]


---

## 4. Green Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| [Short Title] | [Descriptive name] | [Input data or preconditions] | [Expected result] | automated (unit/integ) / e2e / manual verification|
| [Short Title] | [Descriptive name] | [Input data or preconditions] | [Expected result] | automated (unit/integ) / e2e / manual verification|

---

## 5. Red Path Tests

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| [Short Title] | [Descriptive name] | [Invalid/edge-case input] | [Expected error or behavior] | automated (unit/integ) / e2e / manual verification|
| [Short Title] | [Descriptive name] | [Invalid/edge-case input] | [Expected error or behavior] | automated (unit/integ) / e2e / manual verification|

---

## 6. Edge Cases & Boundary Conditions

| Short Title | Test Name | Input / Setup | Expected Outcome | Type |
|---|---|---|---|---|
| [Short Title] | [Descriptive name] | [Boundary value or unusual condition] | [Expected result] | unit / e2e / manual verification|
| [Short Title] | [Descriptive name] | [Boundary value or unusual condition] | [Expected result] | unit / e2e / manual verification|

---

## 9. Existing Automated Tests

- [List any existing test functions/files that cover related behavior, or state "None"]
- Example: `TestParseOpsFlags` in `internal/flag_parser_test.go` (line 8) — 13 subtests

---

## 10. Missing Automated Tests

| Scenario | Type | What It Validates |
|---|---|---|
| [Describe untested scenario] | unit / e2e | [What gap this test would fill] |
| [Describe untested scenario] | unit / e2e | [What gap this test would fill] |

---

## Notes
- [Any additional context, dependencies on other features, or open questions]
