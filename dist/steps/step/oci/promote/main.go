package main

import (
	"log/slog"
	"os"

	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/internal"
)

func main() {
	if err := run(os.Args[1:], os.Getenv); err != nil {
		slog.Error("promote", "err", err)
		os.Exit(1)
	}
}

func run(args []string, getEnv internal.GetEnv) error {
	inputs, err := internal.ParseInputs(args, getEnv)
	if err != nil {
		return err
	}

	slog.SetLogLoggerLevel(inputs.LogLevel)
	return nil
}
