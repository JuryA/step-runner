package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

var _ runner.Cache = &cache{}

type cache struct {
	cacheDir string

	gitFetcher *git.GitFetcher
}

func New() (runner.Cache, error) {
	cacheDir := filepath.Join(os.TempDir(), "step-runner-cache")
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return nil, fmt.Errorf("making cache dir %q: %w", cacheDir, err)
	}

	return &cache{
		cacheDir:   cacheDir,
		gitFetcher: git.New(cacheDir),
	}, nil
}

func (c *cache) Get(ctx context.Context, parentDir string, stepRef *proto.Step_Reference) (*proto.SpecDefinition, error) {
	load := func(dir string) (*proto.SpecDefinition, error) {
		path := filepath.Join(stepRef.Path...)
		filename := filepath.Join(dir, path, stepRef.Filename)
		spec, step, err := schema.LoadSteps(filename)
		if err != nil {
			return nil, fmt.Errorf("loading file %q: %w", filename, err)
		}
		protoSpec, err := spec.Compile()
		if err != nil {
			return nil, fmt.Errorf("compiling file %q: %w", dir, err)
		}
		protoDef, err := step.Compile()
		if err != nil {
			return nil, fmt.Errorf("compiling file %q: %w", dir, err)
		}
		protoStepDef := &proto.SpecDefinition{
			Spec:       protoSpec,
			Definition: protoDef,
		}
		protoStepDef.Dir = filepath.Join(dir, path)
		return protoStepDef, nil
	}

	switch {
	case stepRef.Protocol == proto.StepReferenceProtocol_local:
		return load(parentDir)

	case stepRef.Protocol == proto.StepReferenceProtocol_git:
		dir, err := c.gitFetcher.Get(ctx, stepRef.Url, stepRef.Version)
		if err != nil {
			return nil, fmt.Errorf("fetching step %q: %w", stepRef, err)
		}

		return load(dir)

	default:
		return nil, fmt.Errorf("invalid step reference: %v", stepRef)
	}
}
