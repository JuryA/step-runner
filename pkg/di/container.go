package di

import (
	"fmt"
	"os"
	"path/filepath"

	stepdist "gitlab.com/gitlab-org/step-runner/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/api/service"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

/*
Container provides Dependency Injection (DI) for the step-runner.

DI is where an object's dependencies are passed to it, rather than being created internally.
Promotes loose coupling, testability, and maintainability.
Ideally, constructors are called only in two places: DI, and test builders. This makes it easy to change
the constructor function signature.

For example, creating dependencies within an object:

- OCIFetcher depends on `internal.NewClient` and `cache.New`
- Changing `internal.NewClient` or `cache.New` function signatures forces a change in OCIFetcher

	func NewOCIFetcher() *OCIFetcher {
		return &OCIFetcher{
			client: internal.NewClient(cache.New()),
		}
	}

Creating dependencies using DI:

- OCIFetcher only depends on internal.Client (even better, an interface of Client)
- Changing NewClient or cache.New requires no change in OCIFetcher
- Requires DI Container.OCIFetcher()

	func NewOCIFetcher(client *internal.Client) *OCIFetcher {
		return &OCIFetcher{ client: client }
	}
*/
type Container struct{}

func NewContainer() *Container {
	return &Container{}
}

func (c *Container) StepParser() (*runner.Parser, error) {
	gitFetcher, err := c.GitFetcher()
	if err != nil {
		return nil, fmt.Errorf("creating step parser: %w", err)
	}

	ociFetcher, err := c.OCIFetcher()
	if err != nil {
		return nil, fmt.Errorf("creating step parser: %w", err)
	}

	return runner.NewParser(gitFetcher, ociFetcher, c.DistFetcher()), nil
}

func (c *Container) CacheDir() (string, error) {
	cacheDir := filepath.Join(os.TempDir(), "step-runner-cache")
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return "", fmt.Errorf("creating cache dir %q: %w", cacheDir, err)
	}

	return cacheDir, nil
}

func (c *Container) GitFetcher() (*git.GitFetcher, error) {
	cacheDir, err := c.CacheDir()
	if err != nil {
		return nil, fmt.Errorf("creating git fetcher: %w", err)
	}

	return git.New(cacheDir, git.CloneOptions{Depth: 1}), nil
}

func (c *Container) OCIFetcher() (*oci.OCIFetcher, error) {
	cacheDir, err := c.CacheDir()
	if err != nil {
		return nil, fmt.Errorf("creating oci fetcher: %w", err)
	}

	return oci.NewOCIFetcher(cacheDir), nil
}

func (c *Container) DistFetcher() *dist.Fetcher {
	return dist.NewFetcher(stepdist.FindDistributedStep)
}

func (c *Container) StepRunnerService(env *runner.Environment) (*service.StepRunnerService, error) {
	stepParser, err := c.StepParser()
	if err != nil {
		return nil, fmt.Errorf("creating step runner service: %w", err)
	}

	return service.New(stepParser, env), nil
}
