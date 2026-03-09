package internal

import (
	"fmt"
	"os"
	"strings"
)

// ResolvedLine holds a single shell line after resolution, with per-line
// metadata. Silent is true when the original Opsfile line had a leading @
// prefix, indicating that the executor should not echo it before running.
// IgnoreError is true when the original Opsfile line had a leading - prefix,
// indicating that the executor should ignore non-zero exit codes for this line.
type ResolvedLine struct {
	Text        string
	Silent      bool
	IgnoreError bool
}

// ResolvedCommand holds the shell lines for a command after environment
// selection and variable substitution.
type ResolvedCommand struct {
	Lines []ResolvedLine
}

// Resolve selects the correct environment block for commandName and substitutes
// all variable references in the shell lines.
//
// Environment selection:
//  1. Exact match on env (e.g. "prod")
//  2. "default" block, if the env-specific block is absent
//  3. Error
//
// Variable substitution: $(VAR_NAME) tokens whose content is a bare identifier
// are resolved against vars and envFileVars using a six-level priority chain
// (shell wins — matches Docker Compose / Terraform convention):
//  1. Shell env-scoped:    os.LookupEnv("env_VAR")
//  2. Opsfile env-scoped:  vars["env_VAR"]
//  3. Env-file env-scoped: envFileVars["env_VAR"]
//  4. Shell unscoped:      os.LookupEnv("VAR")
//  5. Opsfile unscoped:    vars["VAR"]
//  6. Env-file unscoped:   envFileVars["VAR"]
//
// $(…) tokens containing spaces or other non-identifier characters are passed
// through unchanged, preserving shell subcommand syntax.
func Resolve(commandName, env string, commands map[string]OpsCommand, vars, envFileVars OpsVariables) (ResolvedCommand, error) {
	cmd, ok := commands[commandName]
	if !ok {
		return ResolvedCommand{}, fmt.Errorf("command %q not found", commandName)
	}

	raw, err := selectLines(cmd, env)
	if err != nil {
		return ResolvedCommand{}, err
	}

	lines := make([]ResolvedLine, 0, len(raw))
	for _, line := range raw {
		silent := false
		ignoreError := false
		for len(line) > 0 {
			switch line[0] {
			case '@':
				if !silent {
					silent = true
					line = line[1:]
					continue
				}
			case '-':
				if !ignoreError {
					ignoreError = true
					line = line[1:]
					continue
				}
			}
			break
		}
		substituted, err := substituteVars(line, env, vars, envFileVars)
		if err != nil {
			return ResolvedCommand{}, err
		}
		lines = append(lines, ResolvedLine{Text: substituted, Silent: silent, IgnoreError: ignoreError})
	}
	return ResolvedCommand{Lines: lines}, nil
}

// selectLines returns the shell lines for env, falling back to "default".
func selectLines(cmd OpsCommand, env string) ([]string, error) {
	if lines, ok := cmd.Environments[env]; ok {
		return lines, nil
	}
	if lines, ok := cmd.Environments["default"]; ok {
		return lines, nil
	}
	return nil, fmt.Errorf("command %q has no environment %q and no default", cmd.Name, env)
}

// substituteVars replaces $(VAR_NAME) references in a single shell line.
// Only tokens whose content passes isIdentifier are treated as Opsfile
// variable references; all others are left unchanged.
func substituteVars(line, env string, vars, envFileVars OpsVariables) (string, error) {
	var b strings.Builder
	remaining := line
	for {
		start := strings.Index(remaining, "$(")
		if start == -1 {
			b.WriteString(remaining)
			break
		}
		b.WriteString(remaining[:start])
		remaining = remaining[start+2:]

		end := strings.Index(remaining, ")")
		if end == -1 {
			// No closing paren — treat "$(" as a literal and continue scanning.
			b.WriteString("$(")
			continue
		}
		token := remaining[:end]
		remaining = remaining[end+1:]

		if isIdentifier(token) {
			val, err := resolveVar(token, env, vars, envFileVars)
			if err != nil {
				return "", err
			}
			b.WriteString(val)
		} else {
			// Not an identifier (e.g. a shell subcommand) — pass through unchanged.
			b.WriteString("$(")
			b.WriteString(token)
			b.WriteString(")")
		}
	}
	return b.String(), nil
}

// resolveVar looks up varName using a six-level priority chain (shell wins):
//  1. Shell env-scoped    (os.LookupEnv("env_VAR"))
//  2. Opsfile env-scoped  (vars["env_VAR"])
//  3. Env-file env-scoped (envFileVars["env_VAR"])
//  4. Shell unscoped      (os.LookupEnv("VAR"))
//  5. Opsfile unscoped    (vars["VAR"])
//  6. Env-file unscoped   (envFileVars["VAR"])
func resolveVar(varName, env string, vars, envFileVars OpsVariables) (string, error) {
	scopedName := env + "_" + varName

	// Priority 1: shell env-scoped
	if val, ok := os.LookupEnv(scopedName); ok {
		return val, nil
	}
	// Priority 2: Opsfile env-scoped
	if val, ok := vars[scopedName]; ok {
		return val, nil
	}
	// Priority 3: env-file env-scoped
	if val, ok := envFileVars[scopedName]; ok {
		return val, nil
	}
	// Priority 4: shell unscoped
	if val, ok := os.LookupEnv(varName); ok {
		return val, nil
	}
	// Priority 5: Opsfile unscoped
	if val, ok := vars[varName]; ok {
		return val, nil
	}
	// Priority 6: env-file unscoped
	if val, ok := envFileVars[varName]; ok {
		return val, nil
	}

	return "", fmt.Errorf("variable %q not defined for environment %q", varName, env)
}
