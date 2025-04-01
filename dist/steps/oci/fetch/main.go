package main

import (
	"log/slog"
	"os"

	"gitlab.com/gitlab-org/dist-steps/oci/fetch/internal"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fetch", "err", err)
		os.Exit(1)
	}
}

func run() error {
	inputs, err := internal.ParseInputs(os.Args[1:], os.Getenv)
	if err != nil {
		return err
	}

	slog.SetLogLoggerLevel(inputs.LogLevel)
	return nil
}
