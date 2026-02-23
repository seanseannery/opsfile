package internal

// the structure of the ops commandline argument should be
// ops [ops arguments] <environment> <command> [command arguments]

type cliArgs struct {
	cliOptions []string
	opsEnv     string
	opsCommand    string
}