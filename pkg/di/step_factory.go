package di

import (
	"gitlab.com/gitlab-org/step-runner/pkg/domain"
)

func InitializeStepFactory() func(*Container) error {
	return func(c *Container) error {
		c.StepFactory = domain.NewStepFactory()
		return nil
	}
}
