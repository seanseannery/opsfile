package main

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)


const opsFileName string = "Opsfile"

func main() {

	args, err := processArgs()
	slog.SetLogLoggerLevel(slog.LevelWarn)

	path, err := getClosestOpsfilePath()
	if err != nil {
		slog.Error("Error getting Opsfile path: " + err.Error())
		return
	}
	print("Found it:" + path)
}

// get opsfile configuration file in the current directory or nearest parent directory
// if it doesn't exist anywhere in the parent tree, return an error
func getClosestOpsfilePath() (string, error) {

	workingDir, wdErr := os.Getwd()
	if wdErr != nil {
		slog.Error("Error getting working directory: " + wdErr.Error())
		return "", wdErr
	}
	slog.Info("Working Directory: " + workingDir)

	currPath := workingDir
	file, err := os.Stat(currPath + string(filepath.Separator) + opsFileName)

	// ignore folders named 'opsfile'
	for (err != nil && os.IsNotExist(err)) || (err == nil && file.IsDir()) {
		slog.Info("Opsfile not found in " + currPath)

		//if the current directory and parent directory are the same, means we are at root and should stop
		if currPath == filepath.Dir(currPath) {
			slog.Error("Could not find Opsfile in any parent directory")
			return "", errors.New("Could not find Opsfile in any parent directory")
		}
		currPath = filepath.Dir(currPath)
		file, err = os.Stat(currPath + string(filepath.Separator) + opsFileName)
	}
	if err != nil {
		slog.Error("Error opening " + currPath + ": " + err.Error())
		return "", err
	}
	return currPath, nil
}

func processArgs() cliArgs {
	return nil
}
