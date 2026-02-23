# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

`opsfile` builds a CLI tool called `ops` — like `make`/`Makefile` but for live operations commands. Users create an `Opsfile` in their repo and run commands as `ops [env] <command> [args]`, e.g. `ops prod tail-logs` or `ops preprod instance-count`.

## Commands

```bash
make build    # build binary to bin/ops
make test     # go test -v ./...
make run      # build and run
make deps     # go mod download && go mod tidy
go test ./internal/...  # run tests for a specific package
```

## Architecture

The project is early-stage. Key design points:

- **Entry point**: `cmd/ops/main.go` — finds the nearest `Opsfile` by walking up the directory tree from cwd (same pattern as git finding `.git`), then parses and executes the requested command.
- **CLI arg structure**: `ops [ops-options] <environment> <command> [command-args]` — defined in `internal/argument_parser.go` as `cliArgs{cliOptions, opsEnv, opsCommand}`.
- **`internal/opsfile_parser.go`**: Intended to parse the `Opsfile` format (currently a stub).
- **`getClosestOpsfilePath()`** in `main.go`: Walks parent directories until it finds a file named `Opsfile` (skips directories with that name). Returns the directory containing the file.

The `docs/` and `examples/` directories exist but are currently empty.

## Module

Module path: `sean_seannery/opsfile` (Go 1.25+). No external dependencies yet.

## Contributing guidelines

Follow `CONTRIBUTING.md` for full details. Key points for Claude:

**Commits** — use [Conventional Commits](https://www.conventionalcommits.org/): `<type>(<scope>): <summary>`. Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`. Subject line under 72 characters, imperative mood.

**Code style** — standard Go conventions: `MixedCaps` exports, short receiver names, errors wrapped with `fmt.Errorf("doing X: %w", err)`, table-driven tests. Run `gofmt` before committing.

**Dependencies** — prefer the standard library; add external packages only when necessary, always follow with `go mod tidy`.
