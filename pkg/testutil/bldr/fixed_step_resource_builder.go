package bldr

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type FixedStepResourceBuilder struct {
	specDef *proto.SpecDefinition
}

func StepResource(specDef *proto.SpecDefinition) *FixedStepResourceBuilder {
	return &FixedStepResourceBuilder{
		specDef: specDef,
	}
}

func (bldr *FixedStepResourceBuilder) Build() runner.StepResource {
	return &FixedStepResource{bldr.specDef}
}

type FixedStepResource struct {
	specDef *proto.SpecDefinition
}

func (sr *FixedStepResource) Describe() string {
	return "fixed-step-resource"
}

func (sr *FixedStepResource) Interpolate(_ *expression.InterpolationContext) (runner.StepResource, error) {
	return sr, nil
}

func (sr *FixedStepResource) Fetch(_ ctx.Context) (*proto.SpecDefinition, error) {
	return sr.specDef, nil
}
