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

const opsFileName string = "Opsfile"

func main() {
	slog.SetLogLoggerLevel(slog.LevelWarn)

	flags, positionals, err := internal.ParseOpsFlags(os.Args[1:], nil)
	if errors.Is(err, internal.ErrHelp) {
		// Best-effort: show available commands alongside help
		if dir, dirErr := resolveOpsfileDir(flags.Directory); dirErr == nil {
			opsfilePath := filepath.Join(dir, opsFileName)
			if _, cmds, cmdOrder, envOrder, perr := internal.ParseOpsFile(opsfilePath); perr == nil {
				fmt.Fprintln(os.Stderr)
				internal.FormatCommandList(os.Stderr, opsfilePath, cmds, cmdOrder, envOrder)
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
		dir, err = getClosestOpsfilePath()
		if err != nil {
			slog.Error("finding Opsfile: " + err.Error())
			os.Exit(1)
		}
	}

	vars, commands, cmdOrder, envOrder, err := internal.ParseOpsFile(filepath.Join(dir, opsFileName))
	if err != nil {
		slog.Error("parsing Opsfile: " + err.Error())
		os.Exit(1)
	}

	if flags.List {
		absPath := filepath.Join(dir, opsFileName)
		displayPath := absPath
		if cwd, cwdErr := os.Getwd(); cwdErr == nil {
			if rel, relErr := filepath.Rel(cwd, absPath); relErr == nil {
				displayPath = rel
			}
		}
		internal.FormatCommandList(os.Stdout, displayPath, commands, cmdOrder, envOrder)
		os.Exit(0)
	}

	args, err := internal.ParseOpsArgs(positionals)
	if err != nil {
		slog.Error("parsing arguments: " + err.Error())
		os.Exit(1)
	}

	resolved, err := internal.Resolve(args.OpsCommand, args.OpsEnv, commands, vars)
	if err != nil {
		slog.Error("resolving command: " + err.Error())
		os.Exit(1)
	}

	if flags.DryRun {
		if !flags.Silent {
			for _, line := range resolved.Lines {
				fmt.Println(line.Text)
			}
		}
		return
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	if err := internal.Execute(resolved.Lines, shell, flags.Silent, os.Stderr); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		slog.Error("executing command: " + err.Error())
		os.Exit(1)
	}
}

// resolveOpsfileDir returns the directory containing the Opsfile, preferring
// flagDir when set and falling back to getClosestOpsfilePath.
func resolveOpsfileDir(flagDir string) (string, error) {
	if flagDir != "" {
		return flagDir, nil
	}
	return getClosestOpsfilePath()
}

// getClosestOpsfilePath returns the directory containing the nearest Opsfile,
// walking up the directory tree from the current working directory.
func getClosestOpsfilePath() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	slog.Info("Working Directory: " + workingDir)

	currPath := workingDir
	file, err := os.Stat(filepath.Join(currPath, opsFileName))

	// ignore folders named 'Opsfile'
	for (err != nil && os.IsNotExist(err)) || (err == nil && file.IsDir()) {
		slog.Info("Opsfile not found in " + currPath)

		if currPath == filepath.Dir(currPath) {
			return "", errors.New("could not find Opsfile in any parent directory")
		}
		currPath = filepath.Dir(currPath)
		file, err = os.Stat(filepath.Join(currPath, opsFileName))
	}
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", currPath, err)
	}
	return currPath, nil
}
