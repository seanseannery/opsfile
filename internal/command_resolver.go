package internal

import (
	"fmt"
	"os"
	"strings"
)

// ResolvedCommand holds the shell lines for a command after environment
// selection and variable substitution.
type ResolvedCommand struct {
	Lines []string
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
// are resolved against vars using env-scoped priority (env_VAR_NAME before
// VAR_NAME). $(…) tokens containing spaces or other non-identifier characters
// are passed through unchanged, preserving shell subcommand syntax.
func Resolve(commandName, env string, commands map[string]OpsCommand, vars OpsVariables) (ResolvedCommand, error) {
	cmd, ok := commands[commandName]
	if !ok {
		return ResolvedCommand{}, fmt.Errorf("command %q not found", commandName)
	}

	raw, err := selectLines(cmd, env)
	if err != nil {
		return ResolvedCommand{}, err
	}

	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		substituted, err := substituteVars(line, env, vars)
		if err != nil {
			return ResolvedCommand{}, err
		}
		lines = append(lines, substituted)
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
func substituteVars(line, env string, vars OpsVariables) (string, error) {
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
			val, err := resolveVar(token, env, vars)
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

// resolveVar looks up varName using a four-level priority chain:
//  1. Opsfile env-scoped  (vars["env_VAR"])
//  2. Shell env-scoped    (os.Getenv("env_VAR"))
//  3. Opsfile unscoped    (vars["VAR"])
//  4. Shell unscoped      (os.Getenv("VAR"))
func resolveVar(varName, env string, vars OpsVariables) (string, error) {
	if val, ok := vars[env+"_"+varName]; ok {
		return val, nil
	}
	if val, ok := os.LookupEnv(env + "_" + varName); ok {
		return val, nil
	}
	if val, ok := vars[varName]; ok {
		return val, nil
	}
	if val, ok := os.LookupEnv(varName); ok {
		return val, nil
	}
	return "", fmt.Errorf("variable %q not defined for environment %q", varName, env)
}
