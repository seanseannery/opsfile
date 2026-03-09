package internal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Execute runs each resolved shell line sequentially using the given shell
// binary. Each command inherits the current process environment and has its
// stdin/stdout/stderr connected to the terminal.
//
// When silent is false, each line's Text is printed to the echo writer before
// execution — unless the line's Silent flag is set (from an @ prefix in the
// Opsfile). When silent is true, no lines are echoed regardless of per-line
// flags.
//
// Returns immediately on the first command failure.
func Execute(lines []ResolvedLine, shell string, silent bool, echo io.Writer) error {
	for _, line := range lines {
		if !silent && !line.Silent {
			fmt.Fprintln(echo, line.Text)
		}
		cmd := exec.Command(shell, "-c", line.Text)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running %q: %w", line.Text, err)
		}
	}
	return nil
}
