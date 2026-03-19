package internal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// DefaultShell returns the user's preferred shell from the SHELL environment
// variable, falling back to /bin/sh if unset.
func DefaultShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}

// Execute runs each resolved shell line sequentially using the given shell
// binary. Each command inherits the current process environment and has its
// stdin/stdout/stderr connected to the terminal.
//
// commandArgs are passed as positional parameters to the shell ($1, $2, $@
// inside the shell script). They are forwarded via the POSIX sh -c convention:
// sh -c "script" -- arg1 arg2 ...
//
// When silent is false, each line's Text is printed to the echo writer before
// execution — unless the line's Silent flag is set (from an @ prefix in the
// Opsfile). When silent is true, no lines are echoed regardless of per-line
// flags.
//
// When dryRun is true, lines are echoed (subject to silent/per-line flags) but
// not executed. Execute returns nil in this case.
//
// When a line's IgnoreError flag is set (from a - prefix in the Opsfile),
// non-zero exit codes from that line are ignored and execution continues.
// System-level errors (e.g., shell not found) still propagate.
//
// Returns immediately on the first non-ignored command failure.
func Execute(lines []ResolvedLine, shell string, silent bool, dryRun bool, commandArgs []string, echo io.Writer) error {
	if dryRun {
		for _, line := range lines {
			if !silent && !line.Silent {
				fmt.Fprintln(echo, line.Text)
			}
		}
		return nil
	}

	for _, line := range lines {
		if !silent && !line.Silent {
			fmt.Fprintln(echo, line.Text)
		}
		args := append([]string{"-c", line.Text, "ops"}, commandArgs...)
		cmd := exec.Command(shell, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if line.IgnoreError && errors.As(err, &exitErr) {
				continue
			}
			return fmt.Errorf("running %q: %w", line.Text, err)
		}
	}
	return nil
}
