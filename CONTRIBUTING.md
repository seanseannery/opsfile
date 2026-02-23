# Contributing

Thanks for your interest in `opsfile`! Contributions are welcome — whether it's a bug fix, a new feature, or just a typo correction. Feel free to submit pull requests with details on the fix or just submit a ticket detailing the feature request or bug fix.

---

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<optional scope>): <short summary>

<optional body>
```

Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

Examples:
```
feat(parser): support multi-line variable values
fix(cli): handle missing Opsfile gracefully
docs: add contributing guide
```

- Keep the subject line under 72 characters
- Use the imperative mood ("add support" not "added support")
- Reference issues in the body if relevant (`Closes #42`)

---

## Pull Requests

- One logical change per PR — keep diffs focused
- PR title should follow the same Conventional Commits format as the commit message
- Include a short description of *why* the change is needed, not just what it does
- Ensure `make test` passes before opening a PR
- Prefer small, reviewable PRs over large ones

---

## Code Review Feedback

Reviewers use [Conventional Comments](https://conventionalcomments.org/) to signal intent:

| Label | Meaning |
|-------|---------|
| `suggestion:` | Optional improvement, not blocking |
| `issue:` | Must be addressed before merge |
| `question:` | Clarification needed, not blocking |
| `nit:` | Minor style/preference, non-blocking |
| `praise:` | Positive callout |

---

## Style and Formatting

- Run `gofmt` (or `goimports`) before committing — CI will catch unformatted code
- Follow standard Go naming conventions: `MixedCaps` for exported identifiers, short receiver names
- Keep functions focused; if a function needs a comment to explain what it does, consider splitting it
- Errors should be wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Prefer table-driven tests using `[]struct{ ... }` subtests

## Dependencies

- Avoid adding external dependencies unless there is a strong reason — the standard library is extensive
- If a dependency is necessary, discuss it in the PR description and ensure it is well-maintained
- Run `go mod tidy` after any dependency changes and commit the updated `go.mod` / `go.sum`
