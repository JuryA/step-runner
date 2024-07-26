package di

import (
	"gitlab.com/gitlab-org/step-runner/pkg/step"
)

func InitializeStepParser() func(*Container) error {
	return func(c *Container) error {
		c.StepParser = step.NewStepParser(c.StepFactory, c.GitFetcher)
		return nil
	}
}
