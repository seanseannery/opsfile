package main

import (
	"fmt"
	"io"
	"strings"

	"sean_seannery/opsfile/internal"
)

// formatCommandList writes a formatted summary of the Opsfile's commands and
// environments to w. Commands are printed in cmdOrder; environments in envOrder.
func formatCommandList(w io.Writer, opsfilePath string, cmds map[string]internal.OpsCommand, cmdOrder []string, envOrder []string) {
	fmt.Fprintf(w, "Commands Found in [%s]:\n", opsfilePath)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Environments:")
	fmt.Fprintf(w, "  %s\n", strings.Join(envOrder, "  "))
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Commands:")

	// Compute max command name length for column alignment.
	maxLen := 0
	for _, name := range cmdOrder {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	for _, name := range cmdOrder {
		cmd := cmds[name]
		if cmd.Description == "" {
			fmt.Fprintf(w, "  %s\n", name)
		} else {
			fmt.Fprintf(w, "  %-*s  %s\n", maxLen, name, cmd.Description)
		}
	}
}
