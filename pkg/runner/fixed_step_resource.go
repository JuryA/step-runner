package runner

import (
	"context"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type FixedStepResource struct {
	specDef *proto.SpecDefinition
}

func NewFixedStepResource(specDef *proto.SpecDefinition) *FixedStepResource {
	return &FixedStepResource{specDef: specDef}
}

func (sr *FixedStepResource) Fetch(_ context.Context, _ *expression.InterpolationContext) (*proto.SpecDefinition, error) {
	return sr.specDef, nil
}
