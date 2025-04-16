package bldr

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type FixedStepResourceBuilder struct {
	specDef *runner.SpecDefinition
}

func StepResource(specDef *runner.SpecDefinition) *FixedStepResourceBuilder {
	return &FixedStepResourceBuilder{
		specDef: specDef,
	}
}

func (bldr *FixedStepResourceBuilder) Build() runner.StepResource {
	return &FixedStepResource{specDef: bldr.specDef}
}

type FixedStepResource struct {
	specDef *runner.SpecDefinition
}

func (sr *FixedStepResource) Fetch(ctx ctx.Context, view *expression.InterpolationContext) (*runner.SpecDefinition, error) {
	return sr.specDef, nil
}
