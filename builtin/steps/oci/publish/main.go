package main

import (
	"context"
	"log/slog"
	"os"

	"gitlab.com/gitlab-org/step-runner/builtin/steps/oci/publish/pkg"
)

func main() {
	logger := slog.Default()

	if err := run(logger); err != nil {
		logger.Error("publish", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	inputs, err := pkg.ParseInputs(os.Args[1:], os.Getenv)
	if err != nil {
		return err
	}

	slog.SetLogLoggerLevel(inputs.LogLevel)

	err = pkg.NewReleaser().Release(context.Background(), inputs.RemoteImageRef, inputs.Common, inputs.PlatformSpecific)
	if err != nil {
		return err
	}

	logger.Info("published step", "image", inputs.RemoteImageRef.MajorMinorPatch().Name())
}
