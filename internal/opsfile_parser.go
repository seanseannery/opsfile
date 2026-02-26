package internal

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// OpsVariables is a flat map of all variables declared in the Opsfile.
// Keys are the raw names as written (e.g. "prod_AWS_ACCOUNT"), values are the raw values.
// Environment-scoped variable resolution is deferred to execution time.
type OpsVariables map[string]string

// OpsCommand represents a single named command with per-environment shell lines.
type OpsCommand struct {
	Name string
	// Environments maps environment name to the ordered list of shell lines to execute.
	// "default" is a valid key used as a fallback at execution time.
	Environments map[string][]string
}

type parseState int

const (
	topLevel      parseState = iota
	inCommand                // inside a command block, looking for environment headers
	inEnvironment            // inside an environment block, collecting shell lines
)

// parser holds the mutable state threaded through the line-by-line scan.
type parser struct {
	variables       OpsVariables
	commands        map[string]*OpsCommand
	state           parseState
	currentCommand  string
	currentEnv      string
	continuationBuf string // accumulated fragments from backslash-continuation lines
	lastShellIndent int    // leading-whitespace count of last new shell line; -1 = none yet
}

// ParseOpsFile reads and parses an Opsfile at the given path.
// It returns the declared variables and commands, skipping comments and blank lines.
func ParseOpsFile(path string) (OpsVariables, map[string]OpsCommand, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening Opsfile: %w", err)
	}
	defer f.Close()

	p := &parser{
		variables:       make(OpsVariables),
		commands:        make(map[string]*OpsCommand),
		lastShellIndent: -1,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := p.processLine(scanner.Text()); err != nil {
			return nil, nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("reading Opsfile: %w", err)
	}

	// Flush any trailing backslash-continuation fragment at end of file.
	p.flushContinuation()

	if err := p.validate(); err != nil {
		return nil, nil, err
	}

	commands := make(map[string]OpsCommand, len(p.commands))
	for k, v := range p.commands {
		commands[k] = *v
	}
	return p.variables, commands, nil
}

// processLine categorises a raw line and dispatches to the appropriate handler.
func (p *parser) processLine(raw string) error {
	isIndented := len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\t')
	line := strings.TrimSpace(raw)

	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}
	// A non-indented line always resets context back to the top level.
	if !isIndented {
		p.flushContinuation()
		p.state = topLevel
	}

	switch p.state {
	case topLevel:
		return p.handleTopLevel(line)
	case inCommand:
		return p.handleInCommand(line)
	case inEnvironment:
		p.handleInEnvironment(line, leadingWhitespace(raw))
	}
	return nil
}

func (p *parser) handleTopLevel(line string) error {
	switch {
	case strings.Contains(line, "="):
		return p.parseVariable(line)
	case strings.HasSuffix(line, ":"):
		return p.startCommand(strings.TrimSuffix(line, ":"))
	}
	return nil
}

func (p *parser) handleInCommand(line string) error {
	if isEnvHeader(line) {
		p.startEnv(strings.TrimSuffix(line, ":"))
	}
	return nil
}

func (p *parser) handleInEnvironment(line string, rawIndent int) {
	switch {
	case isEnvHeader(line):
		p.flushContinuation()
		p.startEnv(strings.TrimSuffix(line, ":"))

	case strings.HasSuffix(line, `\`):
		// Backslash continuation: strip \ and accumulate the fragment.
		p.continuationBuf += strings.TrimSuffix(line, `\`)

	case p.continuationBuf != "":
		// Final line of a backslash-continuation chain.
		p.appendShellLine(p.continuationBuf + line)
		p.continuationBuf = ""
		p.lastShellIndent = rawIndent

	case p.lastShellIndent >= 0 && rawIndent > p.lastShellIndent:
		// Indent-based continuation: join to the previous shell line with a space.
		p.joinLastShellLine(" " + line)

	default:
		// Regular new shell line.
		p.appendShellLine(line)
		p.lastShellIndent = rawIndent
	}
}

func (p *parser) parseVariable(line string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return nil
	}
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return errors.New("variable assignment missing name")
	}
	value, err := extractVariableValue(strings.TrimSpace(parts[1]))
	if err != nil {
		return err
	}
	p.variables[name] = value
	return nil
}

// extractVariableValue strips an inline comment from a raw variable value.
// If the value is quoted (double or single quotes), the quotes are removed and
// the inner content is returned as-is — preserving any '#' inside.
// An unclosed opening quote is an error.
// For unquoted values, a '#' preceded by any whitespace (spaces or tabs) starts
// a comment; everything from that whitespace onwards is stripped.
func extractVariableValue(raw string) (string, error) {
	if len(raw) >= 1 {
		quote := raw[0]
		if quote == '"' || quote == '\'' {
			if end := strings.IndexByte(raw[1:], quote); end >= 0 {
				return raw[1 : end+1], nil
			}
			return "", fmt.Errorf("unclosed %c in variable value", quote)
		}
	}
	// Unquoted: strip trailing inline comment.
	// '#' is a comment start only when immediately preceded by whitespace.
	if idx := indexComment(raw); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), nil
	}
	return raw, nil
}

// indexComment returns the index of the first '#' that is immediately preceded
// by a space or tab, or -1 if no such '#' exists.
func indexComment(s string) int {
	for i := 1; i < len(s); i++ {
		if s[i] == '#' && (s[i-1] == ' ' || s[i-1] == '\t') {
			return i
		}
	}
	return -1
}

func (p *parser) startCommand(name string) error {
	if name == "" {
		return nil
	}
	if _, exists := p.commands[name]; exists {
		return fmt.Errorf("duplicate command %q", name)
	}
	p.currentCommand = name
	p.commands[name] = &OpsCommand{Name: name, Environments: make(map[string][]string)}
	p.state = inCommand
	return nil
}

func (p *parser) startEnv(name string) {
	p.currentEnv = name
	cmd := p.commands[p.currentCommand]
	if _, exists := cmd.Environments[name]; !exists {
		cmd.Environments[name] = []string{}
	}
	p.lastShellIndent = -1
	p.state = inEnvironment
}

func (p *parser) appendShellLine(line string) {
	cmd := p.commands[p.currentCommand]
	cmd.Environments[p.currentEnv] = append(cmd.Environments[p.currentEnv], line)
}

// flushContinuation appends any buffered backslash-continuation fragments as a
// complete shell line and clears the buffer. No-op if the buffer is empty.
func (p *parser) flushContinuation() {
	if p.continuationBuf != "" {
		p.appendShellLine(p.continuationBuf)
		p.continuationBuf = ""
	}
}

// joinLastShellLine appends suffix to the last shell line in the current environment.
func (p *parser) joinLastShellLine(suffix string) {
	lines := p.commands[p.currentCommand].Environments[p.currentEnv]
	if len(lines) > 0 {
		lines[len(lines)-1] += suffix
	}
}

// validate checks post-parse invariants.
func (p *parser) validate() error {
	if len(p.commands) == 0 && len(p.variables) == 0 {
		return errors.New("Opsfile is empty")
	}
	return nil
}

// leadingWhitespace returns the number of leading space/tab characters in s.
func leadingWhitespace(s string) int {
	return len(s) - len(strings.TrimLeft(s, " \t"))
}

// isEnvHeader reports whether line is an environment header (e.g. "prod:").
func isEnvHeader(line string) bool {
	return strings.HasSuffix(line, ":") && isIdentifier(strings.TrimSuffix(line, ":"))
}

// isIdentifier reports whether s is a bare identifier (letters, digits, hyphens, underscores).
// Used to distinguish environment headers from shell lines that happen to end with ":".
func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !isIdentChar(c) {
			return false
		}
	}
	return true
}

func isIdentChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}
