package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/dist/steps/oci/fetch/internal"
)

func main() {
	if err := run(os.Args[1:], os.Getenv); err != nil {
		slog.Error("fetch", "err", err)
		os.Exit(1)
	}
}

func run(args []string, getEnv internal.GetEnv) error {
	inputs, err := internal.ParseInputs(args, getEnv)
	if err != nil {
		return err
	}

	slog.SetLogLoggerLevel(inputs.LogLevel)

	cacheDir := filepath.Join(os.TempDir(), "step-runner-cache")
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return fmt.Errorf("making download dir: %w", err)
	}

	downloadDir, err := internal.NewClient(cacheDir).Pull(context.Background(), inputs.RemoteImageRef)
	if err != nil {
		return err
	}

	slog.Info("fetched step", "image", inputs.RemoteImageRef.String())
	return internal.NewOutputs(inputs.OutputFile).Write(downloadDir, inputs.RemoteImageRef, inputs.StepPath)
}
