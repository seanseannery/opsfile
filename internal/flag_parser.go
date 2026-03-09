package internal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	pflag "github.com/spf13/pflag"
)

// the structure of the ops commandline argument should be
// ops [ops flags] <environment> <command> [command arguments]

// ErrHelp is returned by ParseOpsFlags when -h, --help, or -? is passed.
var ErrHelp = errors.New("help requested")

// OpsFlags holds the values of ops-level flags parsed from the command line.
type OpsFlags struct {
	Directory string // -D / --directory
	EnvFile   string // -e / --env-file
	DryRun    bool   // -d / --dry-run
	List      bool   // -l / --list
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
// Returns ErrHelp if -h, --help, or -? is passed (usage is printed to usageOutput).
// Returns an error for unrecognised flags.
// If usageOutput is nil, os.Stderr is used.
func ParseOpsFlags(osArgs []string, usageOutput io.Writer) (OpsFlags, []string, error) {
	if usageOutput == nil {
		usageOutput = os.Stderr
	}
	fs := pflag.NewFlagSet("ops", pflag.ContinueOnError)
	fs.SetOutput(usageOutput)
	fs.SetInterspersed(false) // stop flag parsing at the first non-flag arg (stdlib flag behaviour)

	dir := fs.StringP("directory", "D", "", "use Opsfile in the given `directory`")
	dryRun := fs.BoolP("dry-run", "d", false, "print commands without executing")
	envFile := fs.StringP("env-file", "e", "", "load variables from env `file` (default: .ops_secrets.env if present)")
	silent := fs.BoolP("silent", "s", false, "execute without printing output")
	list := fs.BoolP("list", "l", false, "list available commands and environments")
	ver := fs.BoolP("version", "v", false, "print the ops version and exit")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "ops version %s (commit: %s) %s/%s\n\n", Version, Commit, runtime.GOOS, runtime.GOARCH)
		fmt.Fprint(fs.Output(), `The 'ops' command runs commonly-used live-operation commands that you define for a specific development or production environment.
It locates the 'Opsfile' in this directory (or the nearest parent directory) and runs the commands that you define in that file.

Usage: ops [flags] <environment> <command> [command-args]
      ex. 'ops preprod open-dashboard' or 'ops --dry-run prod tail-logs'

Note: All flags (including -e) must appear before the environment and command
      arguments. --dry-run will print resolved commands including secret values
      loaded from env files.

Flags:`)
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}

	// Strip all help tokens (-h, --help, -?) before fs.Parse so that pflag
	// does not short-circuit and skip flags that appear after the help flag
	// (e.g. "--help -D /path" must still parse -D).
	helpRequested := false
	filtered := make([]string, 0, len(osArgs))
	envFileCount := 0
	for _, a := range osArgs {
		switch a {
		case "-h", "--help", "-?":
			helpRequested = true
		default:
			if a == "-e" || a == "--env-file" || strings.HasPrefix(a, "--env-file=") {
				envFileCount++
			}
			filtered = append(filtered, a)
		}
	}

	if envFileCount > 1 {
		return OpsFlags{}, nil, errors.New("-e / --env-file may only be specified once")
	}

	if err := fs.Parse(filtered); err != nil {
		return OpsFlags{}, nil, err
	}

	flags := OpsFlags{Directory: *dir, EnvFile: *envFile, DryRun: *dryRun, List: *list, Silent: *silent, Version: *ver}

	if helpRequested {
		fs.Usage()
		return flags, nil, ErrHelp
	}

	return flags, fs.Args(), nil
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
