# Feature: Opsfile Discovery


## 1. Problem Statement & High-Level Goals

### Problem
When a user runs `ops` from any directory within a project, the tool needs to locate the project's `Opsfile` — similar to how `git` finds a `.git` directory by walking up the directory tree. Without this, users would need to always run `ops` from the exact directory containing the Opsfile, which is inconvenient for deeply nested project structures.

### Goals
- [x] Automatically locate the nearest `Opsfile` by walking up the directory tree from `cwd`
- [x] Skip directories named `Opsfile` (only match regular files)
- [x] Provide a `-D` / `--directory` flag to bypass discovery and specify an explicit directory
- [x] Exit with a clear error if no `Opsfile` is found in any ancestor directory

### Non-Goals
- Discovery does not search child/sibling directories — only the current directory and its ancestors
- Discovery does not support alternative filenames (e.g., `opsfile`, `.opsfile`)

---

## 2. Functional Requirements

### FR-1: Parent Directory Walk
When the user runs `ops`, the tool starts in `os.Getwd()` and checks for a file named `Opsfile`. If not found, it moves to the parent directory (`filepath.Dir`) and repeats. The walk terminates at the filesystem root.

### FR-2: Skip Directories Named Opsfile
If an entry named `Opsfile` exists but is a directory (`file.IsDir()`), it is ignored and the walk continues upward.

### FR-3: Directory Flag Override
The `-D` / `--directory` flag allows the user to specify a directory directly, bypassing the discovery walk entirely. When set, the tool uses the Opsfile in the specified directory.

### FR-4: Error on Not Found
If the walk reaches the filesystem root without finding an Opsfile, the tool exits with the error message: `"could not find Opsfile in any parent directory"`.

### Example Usage

Running `ops` from a subdirectory that doesn't contain an Opsfile:
```bash
$ cd /project/src/deep/nested
$ ops deploy
# discovers /project/Opsfile and runs the deploy command
```

Overriding discovery with `-D`:
```bash
$ ops -D /other/project deploy
# uses /other/project/Opsfile directly
```

Error when no Opsfile exists:
```bash
$ cd /tmp/empty
$ ops deploy
ERROR finding Opsfile: could not find Opsfile in any parent directory
```

---

## 3. Non-Functional Requirements

| ID | Category | Requirement | Notes |
|----|----------|-------------|-------|
| NFR-1 | Performance | Discovery should complete in negligible time relative to command execution | Directory walks are bounded by filesystem depth |
| NFR-2 | Compatibility | Works on Linux, macOS, and Windows | Uses `filepath` for cross-platform path handling |
| NFR-3 | Reliability | Graceful error for missing Opsfile and stat failures | Wraps errors with `fmt.Errorf` context |
| NFR-4 | Maintainability | Test coverage for found-in-cwd, found-in-parent, not-found, and directory-skip cases | Tests in `cmd/ops/main_test.go` |

---

## 4. Architecture & Implementation Proposal

### Overview
Discovery is implemented as a single function `getClosestOpsfilePath()` in `cmd/ops/main.go`. A helper function `resolveOpsfileDir()` wraps the logic to prefer the `-D` flag when set. The constant `opsFileName` (`"Opsfile"`) defines the target filename.

### Component Design
- **`getClosestOpsfilePath()`** — Core discovery logic. Walks the directory tree upward using `os.Stat` and `filepath.Dir`.
- **`resolveOpsfileDir(flagDir)`** — Thin wrapper that returns `flagDir` if non-empty, otherwise delegates to `getClosestOpsfilePath()`.
- **`main()`** — Calls `resolveOpsfileDir` (or `getClosestOpsfilePath` directly when the `-D` flag is not set) and passes the resulting directory to the parser pipeline.

### Data Flow
```
os.Getwd() -> stat(currPath/Opsfile) -> [found file?] -> return directory
                    |                         |
                    v (not found / is dir)     |
              filepath.Dir(currPath)          |
                    |                         |
                    v (at root?)              |
              return error  <-----------------+
```

### Key Design Decisions
- **Walk upward, not downward:** Matches the git/make convention where the tool is usable from any subdirectory within a project. Upward-only search is deterministic and fast.
- **Skip directories named Opsfile:** Prevents false positives if a user happens to have a directory with that name.
- **Single-function implementation:** The discovery logic is simple enough that it doesn't warrant its own package or file.

### Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `cmd/ops/main.go` | Existing | Contains `getClosestOpsfilePath()`, `resolveOpsfileDir()`, and the `opsFileName` constant |
| `cmd/ops/main_test.go` | Existing | Tests for found-in-cwd, found-in-parent, not-found, and directory-named-Opsfile-skipped |

---

## 5. Alternatives Considered

### Alternative A: Search Downward Into Subdirectories

**Description:** Instead of walking upward, search the current directory and its children for an Opsfile.

**Pros:**
- Could find Opsfiles in nested project structures

**Cons:**
- Non-deterministic when multiple Opsfiles exist in different subdirectories
- Slower — recursive directory traversal
- Breaks the established convention used by git, make, and similar tools

**Why not chosen:** Upward search is the standard pattern for project-root discovery tools and provides deterministic, fast results.

---

### Alternative B: Separate Discovery Package

**Description:** Extract discovery into its own `internal/discovery` package with a dedicated interface.

**Pros:**
- More testable in isolation
- Could support future discovery strategies (e.g., config files, environment variables)

**Cons:**
- Over-engineered for a single, simple function
- Adds package overhead with no current benefit

**Why not chosen:** KISS — the current single-function implementation is clear and sufficient.

---

## Open Questions
- (none — feature is fully implemented)

---

## 6. Task Breakdown

> *Retrospective — all tasks completed in initial implementation.*

### Phase 1: Foundation
- [x] Define `opsFileName` constant
- [x] Implement `getClosestOpsfilePath()` with upward directory walk
- [x] Implement directory-skip logic for entries named `Opsfile`

### Phase 2: Integration
- [x] Wire `getClosestOpsfilePath()` into `main()` pipeline
- [x] Add `-D` / `--directory` flag to bypass discovery
- [x] Add `resolveOpsfileDir()` helper for flag-or-discovery logic

### Phase 3: Polish
- [x] Write unit tests (found-in-cwd, found-in-parent, not-found, directory-skip)
- [x] Clear error messaging when Opsfile not found

---
