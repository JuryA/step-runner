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

var _ runner.Cache = &cache{}

type cache struct {
	gitFetcher *git.GitFetcher
	ociFetcher *oci.OCIFetcher
}

func New() (runner.Cache, error) {
	cacheDir := filepath.Join(os.TempDir(), "step-runner-cache")
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return nil, fmt.Errorf("making cache dir %q: %w", cacheDir, err)
	}

	gitFetcher := git.New(cacheDir, git.CloneOptions{Depth: 1})
	ociFetcher := oci.NewOCIFetcher(cacheDir)
	return NewWithOptions(WithGitFetcher(gitFetcher), WithOCIFetcher(ociFetcher)), nil
}

func WithGitFetcher(fetcher *git.GitFetcher) func(*cache) {
	return func(c *cache) {
		c.gitFetcher = fetcher
	}
}

func WithOCIFetcher(fetcher *oci.OCIFetcher) func(*cache) {
	return func(c *cache) {
		c.ociFetcher = fetcher
	}
}

func NewWithOptions(options ...func(*cache)) runner.Cache {
	c := &cache{}

	for _, opt := range options {
		opt(c)
	}

	return c
}

func (c *cache) Get(ctx context.Context, parentDir string, stepResource runner.StepResource) (*proto.SpecDefinition, error) {
	stepRef := stepResource.ToProtoStepRef()

	switch {
	case stepRef.Protocol == proto.StepReferenceProtocol_local:
		return c.load(stepRef, parentDir)

	case stepRef.Protocol == proto.StepReferenceProtocol_git:
		dir, err := c.gitFetcher.Get(ctx, stepRef.Url, stepRef.Version)
		if err != nil {
			return nil, fmt.Errorf("fetching step %q: %w", stepRef, err)
		}

		return c.load(stepRef, dir)

	case stepRef.Protocol == proto.StepReferenceProtocol_oci:
		dir, err := c.ociFetcher.Fetch(ctx, stepRef.Url, stepRef.Version)
		if err != nil {
			return nil, fmt.Errorf("fetching step %q: %w", stepRef, err)
		}

		return c.load(stepRef, dir)

	default:
		return nil, fmt.Errorf("invalid step reference: %v", stepRef)
	}
}

func (c *cache) load(stepRef *proto.Step_Reference, stepDir string) (*proto.SpecDefinition, error) {
	path := filepath.Join(stepRef.Path...)
	filename := filepath.Join(stepDir, path, stepRef.Filename)

	spec, step, err := schema.LoadSteps(filename)
	if err != nil {
		return nil, fmt.Errorf("loading file %q: %w", filename, err)
	}

	protoSpec, err := spec.Compile()
	if err != nil {
		return nil, fmt.Errorf("compiling file %q: %w", stepDir, err)
	}

	protoDef, err := step.Compile()
	if err != nil {
		return nil, fmt.Errorf("compiling file %q: %w", stepDir, err)
	}

	protoStepDef := &proto.SpecDefinition{
		Spec:       protoSpec,
		Definition: protoDef,
		Dir:        filepath.Join(stepDir, path),
	}
	return protoStepDef, nil
}
