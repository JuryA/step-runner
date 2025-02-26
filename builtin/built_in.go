package builtin

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

//go:embed bin
var builtInSteps embed.FS

type StepFinder func(step string, options ...func(*FindStepsOptions)) (fs.FS, error)

type FindStepsOptions struct {
	FindIn fs.FS
}

func FindBuiltInStep(step string, options ...func(*FindStepsOptions)) (fs.FS, error) {
	defaultOps := &FindStepsOptions{FindIn: builtInSteps}

	for _, opt := range options {
		opt(defaultOps)
	}

	stepSubDir := filepath.Join("bin", filepath.Clean(step))

	if stepSubDir == "bin" || stepSubDir == "." {
		return nil, fmt.Errorf("built-in step %q not found", step)
	}

	_, err := fs.Stat(defaultOps.FindIn, stepSubDir)

	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("built-in step %q not found", step)
	}

	if err != nil {
		return nil, fmt.Errorf("loading built-in step %q: %w", step, err)
	}

	stepFS, err := fs.Sub(defaultOps.FindIn, stepSubDir)
	if err != nil {
		return nil, fmt.Errorf("loading built-in step %q: %w", step, err)
	}

	return stepFS, nil
}

func WithFileSystem(findIn fs.FS) func(*FindStepsOptions) {
	return func(opts *FindStepsOptions) {
		opts.FindIn = findIn
	}
}
