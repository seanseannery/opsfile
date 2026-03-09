package internal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseEnvFile reads and parses a .env-format file at the given path.
// It returns the declared variables as an OpsVariables map.
//
// Supported syntax:
//   - NAME=value lines (same quoting rules as Opsfile variables)
//   - Lines starting with # (after trimming) are comments and skipped
//   - Blank lines are skipped
//   - Env-scoped names (e.g. prod_VAR) are supported
//   - A line with =value but no name is a parse error
func ParseEnvFile(path string) (OpsVariables, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("env-file %q: %w", path, err)
	}
	defer f.Close()

	vars := make(OpsVariables)
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		name, rawVal, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("env-file %q: line %d: variable assignment missing name", path, lineNum)
		}

		value, err := extractVariableValue(strings.TrimSpace(rawVal))
		if err != nil {
			return nil, fmt.Errorf("env-file %q: line %d: %w", path, lineNum, err)
		}

		vars[name] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("env-file %q: %w", path, err)
	}

	return vars, nil
}
