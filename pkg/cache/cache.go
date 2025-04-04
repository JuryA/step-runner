package cache

import (
	"context"
	"fmt"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

var _ runner.Cache = &cache{}

type cache struct {
	gitFetcher  *git.GitFetcher
	ociFetcher  *oci.OCIFetcher
	distFetcher *dist.Fetcher
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

func WithDistFetcher(fetcher *dist.Fetcher) func(*cache) {
	return func(c *cache) {
		c.distFetcher = fetcher
	}
}

func NewWithOptions(options ...func(*cache)) runner.Cache {
	c := &cache{}

	for _, opt := range options {
		opt(c)
	}

	return c
}

func (c *cache) Get(ctx context.Context, stepResource runner.StepResource) (*proto.SpecDefinition, error) {
	switch sr := stepResource.(type) {
	case *runner.FileSystemStepResource:
		return sr.Fetch(ctx)

	case *runner.GitStepResource:
		return sr.Fetch(ctx)

	case *runner.OCIStepResource:
		return sr.Fetch(ctx)

	case *runner.DistStepResource:
		stepRef := sr.ToProtoStepRef()
		dir, err := c.distFetcher.Fetch(stepRef.Path)
		if err != nil {
			return nil, fmt.Errorf("fetching step %q: %w", stepRef, err)
		}

		stepPath := filepath.Join(stepRef.Path...)
		return c.Get(ctx, runner.NewFileSystemStepResource(filepath.Join(dir, stepPath), stepRef.Filename))
	default:
		return nil, fmt.Errorf("invalid step reference: %s", stepResource.Describe())
	}
}
