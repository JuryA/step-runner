package di

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
)

type Container struct {
	GitFetcher  *git.GitFetcher
	StepFactory step.StepFactory
	StepParser  *step.StepParser
}

func Initialize() (*Container, error) {
	c := &Container{}

	initializers := []func(*Container) error{
		InitializeGitFetcher(),
		InitializeStepFactory(),
		InitializeStepParser(),
	}

	for _, initializer := range initializers {
		if err := initializer(c); err != nil {
			return nil, fmt.Errorf("failed to initialize container: %w", err)
		}
	}

	return c, nil
}
