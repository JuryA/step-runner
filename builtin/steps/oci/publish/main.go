package main

import (
	"context"
	"log/slog"
	"os"

	"gitlab.com/gitlab-org/step-runner/builtin/steps/oci/publish/pkg"
)

func main() {
	logger := slog.Default()

	inputs, err := pkg.ParseInputs(os.Args[1:])
	if err != nil {
		logger.Error("publish", "err", err)
		os.Exit(1)
	}

	if inputs.DebugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	imgRef, err := inputs.ImgRef()
	if err != nil {
		logger.Error("publish", "err", err)
		os.Exit(1)
	}

	releaser := pkg.NewReleaser()
	err = releaser.Release(context.Background(), imgRef, inputs.Common.Add(inputs.PlatformSpecific))
	if err != nil {
		logger.Error("publish", "err", err)
		os.Exit(1)
	}

	logger.Info("published step", "image", imgRef.String())
}
