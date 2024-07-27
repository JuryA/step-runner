package di

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/domain"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
)

type Container struct {
	GlobalCtx   *domain.GlobalCtx
	GitFetcher  *git.GitFetcher
	StepFactory step.StepFactory
	StepParser  *step.StepParser
	CleanUpFns  []func()
}

func Initialize() (*Container, error) {
	c := &Container{
		CleanUpFns: make([]func(), 0),
	}

	initializers := []func(*Container) error{
		InitializeGitFetcher(),
		InitializeStepFactory(),
		InitializeStepParser(),
		InitializeGlobalContext(),
	}

	for _, initializer := range initializers {
		if err := initializer(c); err != nil {
			return nil, fmt.Errorf("failed to initialize container: %w", err)
		}
	}

	return c, nil
}

func (c *Container) CleanUp() {
	for _, cleanUpFn := range c.CleanUpFns {
		cleanUpFn()
	}
}
