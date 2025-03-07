package main

import (
	"context"
	"log/slog"
	"os"

	"gitlab.com/gitlab-org/step-runner/builtin/steps/oci/publish/pkg"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
)

func main() {
	logger := slog.Default()

	inputs, err := pkg.ParseInputs(os.Args[1:])
	if err != nil {
		logger.Error("publish", "err", err)
		os.Exit(1)
	}

	imgRef, err := inputs.ImgRef()
	if err != nil {
		logger.Error("publish", "err", err)
		os.Exit(1)
	}

	releaser := oci.NewReleaser("download_dir")
	err = releaser.Release(context.Background(), imgRef, inputs.Common.Add(inputs.PlatformSpecific))
	if err != nil {
		logger.Error("publish", "err", err)
		os.Exit(1)
	}

	logger.Error("published image", "image", imgRef.String())
}
