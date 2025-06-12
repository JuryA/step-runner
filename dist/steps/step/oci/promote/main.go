package main

import (
	"context"
	"log/slog"
	"os"

	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/internal"
	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/internal/remote"
)

func main() {
	if err := run(os.Args[1:], os.Getenv); err != nil {
		slog.Error("promote", "err", err)
		os.Exit(1)
	}
}

func run(args []string, getEnv internal.GetEnv) error {
	ctx := context.Background()

	inputs, err := internal.ParseInputs(args, getEnv)
	if err != nil {
		return err
	}

	slog.SetLogLoggerLevel(inputs.LogLevel)

	tags, err := remote.ListTags(ctx, inputs.ToImageRef.Repository())
	if err != nil {
		return err
	}

	toRefs, err := inputs.ToImageRef.SemVerRefs(tags)
	if err != nil {
		return err
	}

	return remote.CopyAll(ctx, inputs.FromImageRef, toRefs...)
}
