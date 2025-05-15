package main

import (
	"context"
	"log/slog"
	"os"

	"gitlab.com/gitlab-org/step-runner/dist/steps/oci/build/api"
	"gitlab.com/gitlab-org/step-runner/dist/steps/oci/build/internal"
)

func main() {
	if err := run(os.Args[1:], os.Getenv); err != nil {
		slog.Error("publish", "err", err)
		os.Exit(1)
	}
}

func run(args []string, getEnv internal.GetEnv) error {
	inputs, err := internal.ParseInputs(args, getEnv)
	if err != nil {
		return err
	}

	slog.SetLogLoggerLevel(inputs.LogLevel)

	imageIndex, err := api.NewReleaser().Release(context.Background(), inputs.RemoteImageRef, inputs.Common, inputs.PlatformSpecific)
	if err != nil {
		return err
	}

	slog.Info("published step", "image", inputs.RemoteImageRef.MajorMinorPatch().Name())
	return internal.NewOutputs(inputs.OutputFile).Write(inputs.RemoteImageRef.MajorMinorPatch(), imageIndex)
}
