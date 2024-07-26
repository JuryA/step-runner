package domain

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/domain/resource"
)

type LazilyLoadedStep struct {
	name     string
	resource resource.Resource
}

func NewLazilyLoadedStep(name string, resource resource.Resource) *LazilyLoadedStep {
	return &LazilyLoadedStep{
		name:     name,
		resource: resource,
	}
}

func (lls *LazilyLoadedStep) Run(ctx.Context, *context.Global, *context.Steps) (*StepResult, error) {
	return nil, nil
}
