# Contributing changes to `opsfile`

  Thanks for your interest in `opsfile`! Contributions are welcome — whether it's a bug fix, a new feature, or just a typo correction. Feel free to submit pull requests with details on the fix or just submit an issue ticket detailing the feature request or bug fix. The #1 rule of being a positive contributor to this project is dont be an asshole (non-constructive or non-inclusive negative comments, pushing half-ass code without bothering to follow guidelines, making unnecessary work)

---

# Getting Started

### Setting up your Development Environment

```bash
make setup-local-dev
```
This will install go, as well as git pre-hooks to help ensure you arent pushing bad commits to github

### Building and Running Tests

All common build and test functionality should be provided out of the box with `make` command as defined in `./Makefile`

```bash
  make help       # show the current list of make commands available for build actions
  make build      # build binary to bin/ops
  make release    # build and release versioned binaries to bin/ops (VERSION=1.2.3, COMMIT=ffaa11)
  make run        # build and run the binary
  make clean      # remove build artifacts
  make deps       # download and tidy Go module dependencies
  make test       # run all tests
  make lint       # check formatting (gofmt) and run static analysis (go vet)
```

### Using your Built Binary

After using `make build` or `make release` you can `cd ./bin/` and test the compiled binary  `./ops --help` or whatever arguments you want to try.
See the README.md for examples.  You may want to copy an Opsfile from the `./examples` folder to experiment with or use the -D flag to point at that directory

# Contributing and Code Standards

  Below describes the style guidelines and proper process for making changes in this repo. We use a standard trunk-based development flow 
  1. make a new s feature branch
  2. implement, test and lint your changes 
  3. commit working, tested code to your feature branch and push it to github
  4. create PR against main
  5. if pr test automation passes and the code follows style/formatting, an admin will merge it into `main` branch)
  6. ?
  7. Profit

## Go Code and Style Guidelines

### Style and Formatting

- Follow [Google Go Style Decisions](https://google.github.io/styleguide/go/decisions) to the best of your effort
	- use early returns and indent the error flow, not the happy path
	- use `slices.Contains`, `slices.DeleteFunc`, and the `maps` package instead of manual loops
	- preallocate slices and maps when the size is known: `make([]T, 0, n)`
	- use `map[K]struct{}` for sets, not `map[K]bool`
	- receiver names: single-letter or double-letter abbreviations matching the type (e.g., `sv *Server`, `c *Client`)
	- Follow standard Go naming conventions: `MixedCaps` for exported identifiers, short receiver names
	- Errors should be wrapped with context: `fmt.Errorf("doing X: %w", err)`
	- Prefer table-driven tests using `[]struct{ ... }` subtests for tests covering multiple permutations of input
- K.I.S.S - readability over micro-optimization: clear code is more important than saving microseconds
- Keep functions focused; if a function needs a comment to explain what it does, consider splitting it
- Run `make lint` before committing — CI will catch unformatted code
- Always write automated tests for your contribution. While there is no explict coverage goal, Test coverage should never decrease.

### Error Handling

- wrap errors with `fmt.Errorf("context: %w", err)`, never discard errors silently
- use `errors.Is` / `errors.As` for error checking, not string comparison
- never use `panic` in library code; only in `main` or test helpers
- return `nil` explicitly for the error value on success paths

### Dependencies

- Prefer standard library functions and utilities over reinventing the wheel yourself, the standard library is extensive.
- Only use external dependencies if it improves code simplicity/security and has a very active community, otherwise prefer the standard library
- If a dependency is necessary, discuss it in the PR description and ensure it is well-maintained
- Always `go mod tidy` after any dependency changes and commit the updated `go.mod` / `go.sum`


## Contributing your changes

Pull requests and changes that don't follow the below standards have an extremely likely chance of failing or getting rejected. The installed git hooks will help remind you before you push changes. The github actions in the repo will re-verify and must pass in order to be merged.

### Creating Feature Branches

- Feature branches must be created to stage all of your changes and issue a pull request against main.
- Feature branches must be refreshed using `git pull` to ensure they are up to date and any conflicts are resolved before pushing them to github
- Feature branches must follow the below naming convention:
  - only lowercase alpha-numeric characters, no punctuation outside of hyphen and slash
  - references an issue/ticket id if one exists, otherwise is descriptive and consise (no more than 4 words)
  - prefixes Follow [Conventional Commits](https://www.conventionalcommits.org/):
  	- ex. `feat/add-flag-support`, `fix/resolve-issue-123`, `ci/add-test-job`, `docs/fix-typo-ticket1234`)

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

### Pull Requests

- One logical change per PR — keep diffs focused and prefer small, reviewable PRs over large ones
- PR title should follow the same Conventional Commits format as the commit message
- PR description contents should Include a short description of *why* the change is needed, not just what it does as well as a list of any new dependecies added and tests added.
- Ensure `make test` and `make lint` passes locally before opening a PR or CI automation will auto-reject your PR.

### CI/CD Workflows

`.github/workflows/` contains:
- `release.yml` — manual dispatch only; bumps version tag, builds cross-platform binaries into `bin/`, generates changelog via git-cliff, creates GitHub Release
- `pr_code_check.yml` — triggers on PR; runs `make lint` and `make test`
- `pr_content.yml` — triggers on PR; validates PR title (conventional commit) and PR body (all template sections filled out)
- `version_bump.yml` — auto-triggers on push to main; creates a patch version tag
- `cliff.toml` — git-cliff changelog notes config, lives in `.github/workflows/`

All release binaries are built into `bin/` and named `ops_<platform>_<version>` based on the git tag.

# Code Review Feedback & Community Participation

The #1 rule of being a positive contributor to this project is dont be an asshole (non-constructive or non-inclusive negative comments, pushing half-ass code without bothering to follow guidelines, making unnecessary work).  Breaking this rule will get your account banned.

## Providing PR Feedback / Comments

Reviewers use [Conventional Comments](https://conventionalcomments.org/) to signal intent and provide constructive feedback. Commenters should assume good intent and remember that everyone is at different places in their coding/engineering life-journey:

| Label | Meaning |
|-------|---------|
| `suggestion:` | Optional improvement, not blocking |
| `issue:` | Must be addressed before merge |
| `question:` | Clarification needed, not blocking |
| `nit:` | Minor style/preference, non-blocking |
| `praise:` | Positive callout |

---








