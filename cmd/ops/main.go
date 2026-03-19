package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"sean_seannery/opsfile/internal"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelWarn)

	flags, positionals, err := internal.ParseOpsFlags(os.Args[1:], nil)
	if errors.Is(err, internal.ErrHelp) {
		// Best-effort: show available commands alongside help
		if dir, dirErr := resolveOpsfileDir(flags.Directory); dirErr == nil {
			opsfilePath := filepath.Join(dir, internal.OpsFileName)
			if parsed, perr := internal.ParseOpsFile(opsfilePath); perr == nil {
				fmt.Fprintln(os.Stderr)
				formatCommandList(os.Stderr, opsfilePath, parsed.Commands, parsed.CommandOrder, parsed.EnvOrder)
			}
		}
		os.Exit(0)
	}
	if err != nil {
		slog.Error("parsing flags: " + err.Error())
		os.Exit(1)
	}

	if flags.Version {
		fmt.Printf("ops version %s (commit: %s) %s/%s\n", internal.Version, internal.Commit, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	var dir string
	if flags.Directory != "" {
		dir = flags.Directory
	} else {
		dir, err = internal.FindOpsfileDir()
		if err != nil {
			slog.Error("finding Opsfile: " + err.Error())
			os.Exit(1)
		}
	}

	parsed, err := internal.ParseOpsFile(filepath.Join(dir, internal.OpsFileName))
	if err != nil {
		slog.Error("parsing Opsfile: " + err.Error())
		os.Exit(1)
	}

	envFileVars, err := internal.LoadEnvFile(flags.EnvFile, dir)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if flags.List {
		absPath := filepath.Join(dir, internal.OpsFileName)
		displayPath := absPath
		if cwd, cwdErr := os.Getwd(); cwdErr == nil {
			if rel, relErr := filepath.Rel(cwd, absPath); relErr == nil {
				displayPath = rel
			}
		}
		formatCommandList(os.Stdout, displayPath, parsed.Commands, parsed.CommandOrder, parsed.EnvOrder)
		os.Exit(0)
	}

	args, err := internal.ParseOpsArgs(positionals)
	if err != nil {
		slog.Error("parsing arguments: " + err.Error())
		os.Exit(1)
	}

	resolved, err := internal.Resolve(args.OpsCommand, args.OpsEnv, parsed.Commands, parsed.Variables, envFileVars, os.LookupEnv)
	if err != nil {
		slog.Error("resolving command: " + err.Error())
		os.Exit(1)
	}

	if err := internal.Execute(resolved.Lines, internal.DefaultShell(), flags.Silent, flags.DryRun, args.CommandArgs, os.Stderr); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		slog.Error("executing command: " + err.Error())
		os.Exit(1)
	}
}

// resolveOpsfileDir returns the directory containing the Opsfile, preferring
// flagDir when set and falling back to internal.FindOpsfileDir.
func resolveOpsfileDir(flagDir string) (string, error) {
	if flagDir != "" {
		return flagDir, nil
	}
	return internal.FindOpsfileDir()
}
