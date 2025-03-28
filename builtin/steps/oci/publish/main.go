package main

import (
	"context"
	"log/slog"
	"os"

	"gitlab.com/gitlab-org/step-builtins/oci/publish/internal"
)

func main() {
	logger := slog.Default()

	if err := run(logger); err != nil {
		logger.Error("publish", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	inputs, err := internal.ParseInputs(os.Args[1:], os.Getenv)
	if err != nil {
		return err
	}

	slog.SetLogLoggerLevel(inputs.LogLevel)

	imageIndex, err := internal.NewReleaser().Release(context.Background(), inputs.RemoteImageRef, inputs.Common, inputs.PlatformSpecific)
	if err != nil {
		return err
	}

	logger.Info("published step", "image", inputs.RemoteImageRef.MajorMinorPatch().Name())
	return internal.NewOutputs(inputs.OutputFile).Write(inputs.RemoteImageRef.MajorMinorPatch(), imageIndex)
}
