package internal

import (
	"errors"
	"fmt"
	"runtime"

	pflag "github.com/spf13/pflag"
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
	fs := pflag.NewFlagSet("ops", pflag.ContinueOnError)
	fs.SetInterspersed(false) // stop flag parsing at the first non-flag arg (stdlib flag behaviour)

	dir := fs.StringP("directory", "D", "", "use Opsfile in the given `directory`")
	dryRun := fs.BoolP("dry-run", "d", false, "print commands without executing")
	silent := fs.BoolP("silent", "s", false, "execute without printing output")
	ver := fs.BoolP("version", "v", false, "print the ops version and exit")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "ops version %s (commit: %s) %s/%s\n\n", Version, Commit, runtime.GOOS, runtime.GOARCH)
		fmt.Fprint(fs.Output(), `The 'ops' command runs commonly-used live-operation commands that you define for a specific development or production environment.
It locates the 'Opsfile' in this directory (or the nearest parent directory) and runs the commands that you define in that file.

Usage: ops [flags] <environment> <command> [command-args]
      ex. 'ops preprod open-dashboard' or 'ops --dry-run prod tail-logs'

Flags:`)
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}

	// -? is not a valid flag name; handle it before fs.Parse.
	for _, a := range osArgs {
		if a == "-?" {
			fs.Usage()
			return OpsFlags{}, nil, ErrHelp
		}
	}

	if err := fs.Parse(osArgs); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return OpsFlags{}, nil, ErrHelp
		}
		return OpsFlags{}, nil, err
	}
	return OpsFlags{Directory: *dir, DryRun: *dryRun, Silent: *silent, Version: *ver}, fs.Args(), nil
}

// ParseOpsArgs parses the positional non-flag arguments returned by ParseOpsFlags into
// an Args struct. Returns an error if the environment or command are missing.
func ParseOpsArgs(nonFlagArgs []string) (Args, error) {
	if len(nonFlagArgs) < 1 {
		return Args{}, errors.New("missing environment argument")
	}
	if len(nonFlagArgs) < 2 {
		return Args{}, errors.New("missing command argument")
	}
	return Args{
		OpsEnv:      nonFlagArgs[0],
		OpsCommand:  nonFlagArgs[1],
		CommandArgs: nonFlagArgs[2:],
	}, nil
}
