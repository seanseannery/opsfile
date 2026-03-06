
## Project Overview

  `opsfile` project builds a CLI tool called `ops` — it functions like `make`/`Makefile` but for live operations commands instead of CI/CD commands. Users create an `Opsfile` in their repo and run commands as `ops [env] <command> [args]`, e.g. `ops prod tail-logs` or `ops preprod instance-count`. More details in `README.md` if needed for additional context.

## Commands

  ```bash
    make help       # show the current list of make commands available for build actions
    make build      # build binary to bin/ops
    make release    # build versioned release binaries (VERSION=1.2.3, COMMIT=ffaabb | default: populated fromt tags)
    make run        # build and run the binary
    make clean      # remove build artifacts
    make deps       # download and tidy Go module dependencies
    make test       # run all tests
    make lint       # check formatting (gofmt) and run static analysis (go vet)
  ```

## Directory Structure
  ```
  opsfile/
    ├── cmd/ops/              Entry point — wires all features and internal code to implement 'ops' cli tool
    ├── internal/             All core logic: flag parsing, Opsfile parsing, command resolution, shell execution, and tests
    ├── examples/             Reference Opsfiles — platform-specific examples (aws, k8s, azure, gcp, baremetal, local)
    ├── install/              Curl-pipe shell installer script for end-users
    ├── bin/                  Compiled binary output (gitignored)
    ├── docs/                 Feature requirements and implementation/architecture documentation
    ├────── testplans/        Test plans for each feature (manual and automated)
    ├── .github/              Github Actions and PR/Issue templates
    ├────── workflows/        GitHub Actions workflows definitions (release, PR checks) and cliff.toml changelog config
    ├── .githooks/            Local git hooks: pre-push (lint+test), commit-msg (conventional commit format)
    │
    ├── go.mod                Module declaration (sean_seannery/opsfile, Go 1.25+, no external deps)
    ├── AGENTS.md             This file — source of truth for agentic context
    ├── CLAUDE.md             Links to AGENTS.md (Claude does not natively support AGENTS.md)
    ├── CONTRIBUTING.md       Dev setup, PR process, and community guidelines
    ├── LICENSE               Project licence.
    ├── Makefile              Build, test, lint, release, and local dev setup commands
    └── README.md             User-facing documentation and usage guide
  ```


## Architecture

  **Execution flow** (top to bottom):
  1. `cmd/ops/main.go` — calls `getClosestOpsfilePath()` to find the nearest `Opsfile` by walking parent dirs (same pattern as git), then sequences the pipeline below.
  2. `internal/flag_parser.go` — `ParseOpsFlags(osArgs)` strips ops-level flags (`--dry-run`, `--silent`, `--directory`, `--version`); `ParseOpsArgs(remaining)` extracts `OpsEnv`, `OpsCommand`, and `CommandArgs`.
  3. `internal/opsfile_parser.go` — `ParseOpsFile(path)` reads the Opsfile and returns `OpsVariables` (a `map[string]string`) and `OpsCommands` (a `map[string]OpsCommand`). Each `OpsCommand` holds a map of env-name → shell lines.
  4. `internal/command_resolver.go` — `ResolveCommand(cmd, env, vars)` selects the correct env block (falling back to `default`), then resolves `$(VAR)` references using a four-level priority chain: Opsfile env-scoped → shell env-scoped → Opsfile unscoped → shell unscoped (`os.LookupEnv`). Non-identifier tokens (e.g. `$(shell ...)`) are passed through unchanged.
  5. `internal/executor.go` — executes the resolved shell lines, respecting `--dry-run` and `--silent` flags.

  **Key types** (all in `internal/`):
  - `OpsFlags` — parsed ops-level CLI flags
  - `Args` — `{OpsEnv, OpsCommand, CommandArgs}`
  - `OpsVariables` — `map[string]string` (variable name → value)
  - `OpsCommand` — `map[string][]string` (environment name → shell lines)

  **Version embedding**: `internal/version.go` declares `Version` and `Commit` vars (defaults `"0.0.0-dev"` / `"none"`); overridden at build time via `-ldflags "-X internal.Version=... -X internal.Commit=..."`.

## Agent Behaviour

  ### Contributing Code and Features

  Must Read (if you havent already) and adhere to `CONTRIBUTING.md` for style, design choices, and code contribution guidelines. High priority contribution guidelines include: 
    - Prefer readability over micro-optimization: clear code is more important than saving microseconds
    - Prefer standard library functions and utilities over reimplementing the wheel yourself, the standard library is extensive.
    - Only use external dependencies if it improves code simplicity/security and has a very active community, otherwise prefer the standard library
    - Follow [Google Go Style Decisions](https://google.github.io/styleguide/go/decisions) to the best of your effort
    - Follow trunk-based development flow, using feature branches and conventional commit standards for commit and pr title naming
      - **Commits** — use [Conventional Commits](https://www.conventionalcommits.org/): `<type>(<scope>): <summary>`. Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`. Subject line under 72 characters, imperative mood.
      - Before committing, check the current branch name and verify it is scoped to the work being committed. If the current branch is `main` or is focused on a different feature/topic than the changes being committed, create a new appropriately-named branch first (following the naming conventions in CONTRIBUTING.md) before committing. New branches must always be created off of `main` unless the user explicitly instructs otherwise.


  ### Must Do
  - For changes touching more than one non-test .go file: state the approach before writing code. If it is a new feature, ask if new documentation should be created in /docs folder
  - check the Go version in `go.mod` and use idiomatic features and libraries available at that version
  - always follow external package changes with `go mod tidy`
  - Tests that parse example files in `examples/` must be updated when new example Opsfiles are added.
  - Always check for remote code changes before starting development or committing changes.  Prefer rebase from remote `main` into the current feature branch by using `git pull --rebase origin main`. Dont resolve conflicts, instead prompt the user.

  ### Must NOT Do
  - do not add new heavy dependencies without approval. do not use dependencies that have small, inactive communities or known vulnerabilites.
  - Never lower the quality or coverage of an existing test to make a broken feature work. If a change requires doing this, prompt for approval before proceeding.
  - Never read or load LICENSE file into context unless explicitly asked
  - Do NOT include a `Co-Authored-By` trailer in commit messages or wrap in a HEREDOC unnecessarily
  - Tests must not pin to values that change between releases (version strings, build timestamps). Validate shape/format instead (e.g. semver regex, non-empty check).
  - Never resolve git merge or rebase conflicts without user input.

  ### Should Do
  - If new directories are created or detected. update this AGENTS.md directory structure section.
  - When asked to "commit this", run `make lint` and `make test` first unless no .go files were changed or if the user says otherwise
  - Prefer using `make` commands for build, test, lint activities over direct go cli commands (unless testing smaller, single-file changes).
  - Adhere .github/pull_request_template.md structure if asked to create a pull request
  - When told `commit and push` also create a github pull request describing the change.

  ### Allowed without prompt:
  - read files, list files, `ls`
  - all `make` commands and their go equivalents (`go test`, `go fmt`, `go build`, etc)
  - editing markdown files

  ### Ask first:
  - package installs,
  - git push
  - deleting files, `rm`, chmod

