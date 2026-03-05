# Opsfile Discovery

## Functional Requirements

- When the user runs `ops`, the tool must locate an `Opsfile` in the current working directory or the nearest parent directory
- The search walks upward from `cwd` toward the filesystem root, checking each directory for a file named `Opsfile`
- Directories named `Opsfile` are skipped -- only regular files match
- If no `Opsfile` is found in any ancestor directory, the tool exits with an error
- The user can bypass discovery entirely by passing `-D <directory>` / `--directory <directory>`, which uses the Opsfile in the specified directory instead

## Implementation Overview

Discovery is implemented in `cmd/ops/main.go` in the function `getClosestOpsfilePath()`.

**Data flow:**

1. `os.Getwd()` obtains the current working directory
2. A loop calls `os.Stat(filepath.Join(currPath, "Opsfile"))` at each level
3. If the stat succeeds but the result is a directory (`file.IsDir()`), the entry is skipped and the walk continues upward
4. If the stat fails with `os.IsNotExist`, the path moves to `filepath.Dir(currPath)` (the parent)
5. The loop terminates when the current path equals its own parent (filesystem root) -- at that point an error is returned
6. On success, the directory containing the `Opsfile` is returned to `main()`

**Key files and symbols:**

- `cmd/ops/main.go` -- `getClosestOpsfilePath() (string, error)`
- The constant `opsFileName` (value `"Opsfile"`) defines the target filename
- The `-D` / `--directory` flag in `main()` short-circuits discovery by providing the directory directly
