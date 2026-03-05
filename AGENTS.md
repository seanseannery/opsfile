
## Project Overview

`opsfile` project builds a CLI tool called `ops` — it functions like `make`/`Makefile` but for live operations commands instead of CI/CD commands. Users create an `Opsfile` in their repo and run commands as `ops [env] <command> [args]`, e.g. `ops prod tail-logs` or `ops preprod instance-count`. More details in `README.md` if needed for additional context.

## Commands

```bash
  make help       # show the current list of make commands available for build actions
  make build      # build binary to bin/ops
  make release    # bump version and build release binaries (BUMP=major|minor|patch, default: patch)
  make run        # build and run the binary
  make clean      # remove build artifacts
  make deps       # download and tidy Go module dependencies
  make test       # run all tests
  make lint       # check formatting (gofmt) and run static analysis (go vet)
```

### Core Architecture and Directory Structure
```
 opsfile/                                                                                                                                                                           
  ├── cmd/ops/          Entry point — wires flag parsing, Opsfile discovery, parsing, resolution, and execution                                                                    
  ├── internal/         All core logic: flag parsing, Opsfile parsing, command resolution, shell execution, and tests                                                                
  ├── examples/         Reference Opsfile showing variables, env blocks, and multiline commands                                                                                      
  ├── install/          Curl-pipe shell installer script for end-users                                                                                                                             
  ├── bin/              Compiled binary output (gitignored)
  ├── docs/             Feature requirements and implementation/architecture documentation
  ├──── testplans/      Documentation for steps required to automatically or manually test features.
  │
  ├── go.mod            Module declaration (sean_seannery/opsfile, Go 1.25+, no external deps)
  ├── CLAUDE.md         Links to AGENTS.md since claude doesnt support it natively
  ├── AGENTS.md         This file.  Source of truth for Agentic context
  └── Makefile          CI/CD development scripts including: Build, test, run, deps, and release
```



## Architecture

- **Entry point**: `cmd/ops/main.go` — finds the nearest `Opsfile` by walking up the directory tree from cwd (same pattern as git finding `.git`), then parses and executes the requested command.
- **CLI arg structure**: `ops [ops-options] <environment> <command> [command-args]` — defined in `internal/argument_parser.go` as `cliArgs{cliOptions, opsEnv, opsCommand}`.
- **`internal/opsfile_parser.go`**: Intended to parse the `Opsfile` format (currently a stub).
- **`getClosestOpsfilePath()`** in `main.go`: Walks parent directories until it finds a file named `Opsfile` (skips directories with that name). Returns the directory containing the file.

## Directory Structure

### Module

Module path: `sean_seannery/opsfile` (Go 1.25+). No external dependencies yet.

## Contributing guidelines

Follow `CONTRIBUTING.md` for full details. Key points:


**Commits** — use [Conventional Commits](https://www.conventionalcommits.org/): `<type>(<scope>): <summary>`. Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`. Subject line under 72 characters, imperative mood.

# Go Coding Style Style



### Specific conventions for this project:

- [Google Go Style Decisions](https://google.github.io/styleguide/go/decisions)
- check the Go version in `go.mod` and use idiomatic features available at that version
- readability over micro-optimization: clear code is more important than saving microseconds
- prefer standard library functions and utilities over writing your own
- use early returns and indent the error flow, not the happy path
- use `slices.Contains`, `slices.DeleteFunc`, and the `maps` package instead of manual loops
- preallocate slices and maps when the size is known: `make([]T, 0, n)`
- use `map[K]struct{}` for sets, not `map[K]bool`
- receiver names: single-letter abbreviations matching the type (e.g., `s *Server`, `c *Client`)
- run `go fmt` after modifying Go source files, never indent manually

### Error Handling

- wrap errors with `fmt.Errorf("context: %w", err)`, never discard errors silently
- use `errors.Is` / `errors.As` for error checking, not string comparison
- never use `panic` in library code; only in `main` or test helpers
- return `nil` explicitly for the error value on success paths

**Dependencies** — prefer the standard library; add external packages only when necessary, always follow with `go mod tidy`.
