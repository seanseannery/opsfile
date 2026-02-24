package internal

import (
	"errors"
	"flag"
	"fmt"
)

// the structure of the ops commandline argument should be
// ops [ops flags] <environment> <command> [command arguments]

// ErrHelp is returned by ParseOpsFlags when -h, --help, or -? is passed.
var ErrHelp = errors.New("help requested")

// OpsFlags holds the values of ops-level flags parsed from the command line.
type OpsFlags struct {
	Directory string // -D / --directory
	DryRun    bool   // -d / --dry-run
	Silent    bool   // -s / --silent
	Version   bool   // -v / --version
}

// Args holds the parsed positional components of an ops CLI invocation.
type Args struct {
	OpsEnv      string   // first positional argument (e.g. "prod")
	OpsCommand  string   // second positional argument (e.g. "tail-logs")
	CommandArgs []string // everything after OpsCommand (passed through to the command)
}

// ParseOpsFlags parses ops-level flags from osArgs and returns the flag values
// and the remaining positional arguments.
// Returns ErrHelp if -h, --help, or -? is passed (usage is printed to stderr).
// Returns an error for unrecognised flags.
func ParseOpsFlags(osArgs []string) (OpsFlags, []string, error) {
	// Pre-process: -? is not a valid flag name; map it to -h.
	processed := make([]string, len(osArgs))
	for i, a := range osArgs {
		if a == "-?" {
			processed[i] = "-h"
		} else {
			processed[i] = a
		}
	}

	fs := flag.NewFlagSet("ops", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: ops [flags] <environment> <command> [command-args]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Flags:")
		fmt.Fprintln(fs.Output(), "  -D, --directory  use Opsfile in the given directory")
		fmt.Fprintln(fs.Output(), "  -d, --dry-run    print commands without executing")
		fmt.Fprintln(fs.Output(), "  -h, --help, -?   show this help message")
		fmt.Fprintln(fs.Output(), "  -s, --silent     execute without printing output")
		fmt.Fprintln(fs.Output(), "  -v, --version    print the ops version and exit")
	}

	var f OpsFlags
	fs.StringVar(&f.Directory, "D", "", "")
	fs.StringVar(&f.Directory, "directory", "", "")
	fs.BoolVar(&f.DryRun, "d", false, "")
	fs.BoolVar(&f.DryRun, "dry-run", false, "")
	fs.BoolVar(&f.Silent, "s", false, "")
	fs.BoolVar(&f.Silent, "silent", false, "")
	fs.BoolVar(&f.Version, "v", false, "")
	fs.BoolVar(&f.Version, "version", false, "")

	if err := fs.Parse(processed); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return OpsFlags{}, nil, ErrHelp
		}
		return OpsFlags{}, nil, err
	}
	return f, fs.Args(), nil
}

// ParseOpsArgs parses the positional arguments returned by ParseOpsFlags into
// an Args struct. Returns an error if the environment or command are missing.
func ParseOpsArgs(positionals []string) (Args, error) {
	if len(positionals) < 1 {
		return Args{}, errors.New("missing environment argument")
	}
	if len(positionals) < 2 {
		return Args{}, errors.New("missing command argument")
	}
	return Args{
		OpsEnv:      positionals[0],
		OpsCommand:  positionals[1],
		CommandArgs: positionals[2:],
	}, nil
}
