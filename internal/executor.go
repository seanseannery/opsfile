package internal

import (
	"fmt"
	"os"
	"os/exec"
)

// Execute runs each shell line sequentially using the given shell binary.
// Each command inherits the current process environment and has its
// stdin/stdout/stderr connected to the terminal.
// Returns immediately on the first command failure.
func Execute(lines []string, shell string) error {
	for _, line := range lines {
		cmd := exec.Command(shell, "-c", line)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running %q: %w", line, err)
		}
	}
	return nil
}
