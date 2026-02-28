# Contributing

Thanks for your interest in `opsfile`! Contributions are welcome — whether it's a bug fix, a new feature, or just a typo correction. Feel free to submit pull requests with details on the fix or just submit a ticket detailing the feature request or bug fix.

---

# Getting Started

### Setting up your Development Environment

```bash
brew install go@.25.5
```

### Building and Running Tests

All common build and test functionality should be provided out of the box with `make` command as defined in `./Makefile`

```bash
make build    # build binary to bin/ops
make test     # go test -v ./...
make run      # build and run
make deps     # go mod download && go mod tidy
make release [patch/minor/major] # bump the version number and build binaries
go test ./internal/...  # run tests for a specific package
```

# Contributing and Code Standards

Pull requests and changes that don't follow the below standards have an extremely likely chance of failing or getting rejected.

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):
```
<type>(<optional scope>): <short summary> (issueID)
```
Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`
Examples:
```
fix: handle missing Opsfile gracefully (Issue#42)
feat(parser): support multi-line variable values
docs: add contributing guide
```

### Style and Formatting

- Run `gofmt` (or `goimports`) before committing — CI will catch unformatted code
- Follow standard Go naming conventions: `MixedCaps` for exported identifiers, short receiver names
- Keep functions focused; if a function needs a comment to explain what it does, consider splitting it
- Errors should be wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Prefer table-driven tests using `[]struct{ ... }` subtests

### Dependencies

- Avoid adding external dependencies unless there is a strong reason — the standard library is extensive
- If a dependency is necessary, discuss it in the PR description and ensure it is well-maintained
- Run `go mod tidy` after any dependency changes and commit the updated `go.mod` / `go.sum`

### Pull Requests

- One logical change per PR — keep diffs focused and prefer small, reviewable PRs over large ones
- PR title should follow the same Conventional Commits format as the commit message
- PR description contents should Include a short description of *why* the change is needed, not just what it does as well as a list of any new dependecies added and tests added.
- Ensure `make test` passes locally before opening a PR or CI automation will auto-reject your PR.

### Code Review Feedback

Reviewers use [Conventional Comments](https://conventionalcomments.org/) to signal intent:

| Label | Meaning |
|-------|---------|
| `suggestion:` | Optional improvement, not blocking |
| `issue:` | Must be addressed before merge |
| `question:` | Clarification needed, not blocking |
| `nit:` | Minor style/preference, non-blocking |
| `praise:` | Positive callout |

---

