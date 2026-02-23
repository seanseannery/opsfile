package internal

import (
	"errors"
	"flag"
)

// the structure of the ops commandline argument should be
// ops [ops arguments] <environment> <command> [command arguments]

// Args holds the parsed positional components of an ops CLI invocation.
type Args struct {
	OpsEnv      string   // first positional argument (e.g. "prod")
	OpsCommand  string   // second positional argument (e.g. "tail-logs")
	CommandArgs []string // everything after OpsCommand (passed through to the command)
}

// ParseOpsFlags parses ops-level flags from osArgs and returns the remaining
// positional arguments. Any ops-level flags (e.g. --verbose) can be
// registered on the FlagSet here in future without changing the signature.
// Returns an error for unrecognised flags.
func ParseOpsFlags(osArgs []string) ([]string, error) {
	fs := flag.NewFlagSet("ops", flag.ContinueOnError)
	if err := fs.Parse(osArgs); err != nil {
		return nil, err
	}
	return fs.Args(), nil
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
