package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

const RunnerCacheDirKey = "STEP_RUNNER_CACHE_DIR"

var _ runner.Cache = &cache{}

type cache struct {
	gitFetcher *git.GitFetcher
	ociFetcher *oci.Fetcher
}

func New(opts ...Option) (runner.Cache, error) {
	var options options

	options.dir = os.Getenv(RunnerCacheDirKey)

	for _, o := range opts {
		err := o(&options)
		if err != nil {
			return nil, err
		}
	}

	if options.dir == "" {
		userCacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("getting user cache dir: %w", err)
		}

		options.dir = filepath.Join(userCacheDir, "step-runner-cache")
	}

	if err := os.MkdirAll(options.dir, 0o750); err != nil {
		return nil, fmt.Errorf("making cache dir %q: %w", options.dir, err)
	}

	return &cache{
		gitFetcher: git.New(options.dir, git.CloneOptions{Depth: options.gitDepth}),
		ociFetcher: oci.New(options.dir),
	}, nil
}

func (c *cache) Get(ctx context.Context, parentDir string, stepResource runner.StepResource) (*proto.SpecDefinition, error) {
	stepRef := stepResource.ToProtoStepRef()

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

	switch stepRef.Protocol {
	case proto.StepReferenceProtocol_local:
		return load(parentDir)

	case proto.StepReferenceProtocol_git:
		dir, err := c.gitFetcher.Get(ctx, stepRef.Url, stepRef.Version)
		if err != nil {
			return nil, fmt.Errorf("fetching step (git) %q: %w", stepRef, err)
		}

		return load(dir)

	case proto.StepReferenceProtocol_oci:
		dir, err := c.ociFetcher.Get(ctx, stepRef.Url, stepRef.Version)
		if err != nil {
			return nil, fmt.Errorf("fetching step (oci) %q: %w", stepRef, err)
		}

		return load(dir)

	default:
		return nil, fmt.Errorf("invalid step reference: %v", stepRef)
	}
}
