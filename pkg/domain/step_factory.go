package domain

import "gitlab.com/gitlab-org/step-runner/pkg/domain/resource"

type StepFactory struct {
}

func NewStepFactory() *StepFactory {
	return &StepFactory{}
}

func (sf *StepFactory) CreateLazilyLoadedStep(parser StepParser, name string, resource resource.Resource) Step {
	return NewLazilyLoadedStep(parser, name, resource)
}
