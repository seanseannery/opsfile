package internal

import (
	"errors"
	"flag"
)

// the structure of the ops commandline argument should be
// ops [ops arguments] <environment> <command> [command arguments]

// Args holds the parsed components of an ops CLI invocation.
type Args struct {
	OpsEnv      string   // first positional argument (e.g. "prod")
	OpsCommand  string   // second positional argument (e.g. "tail-logs")
	CommandArgs []string // everything after OpsCommand (passed through to the command)
}

// ParseArgs parses os.Args[1:] into an Args struct.
// Flags (tokens starting with "-") are handled by a flag.FlagSet; any
// ops-level flags can be registered on the FlagSet in future without
// changing this function's signature.
func ParseArgs(osArgs []string) (Args, error) {
	fs := flag.NewFlagSet("ops", flag.ContinueOnError)
	if err := fs.Parse(osArgs); err != nil {
		return Args{}, err
	}
	pos := fs.Args()
	if len(pos) < 1 {
		return Args{}, errors.New("missing environment argument")
	}
	if len(pos) < 2 {
		return Args{}, errors.New("missing command argument")
	}
	return Args{
		OpsEnv:      pos[0],
		OpsCommand:  pos[1],
		CommandArgs: pos[2:],
	}, nil
}
