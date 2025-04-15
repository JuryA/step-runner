package runner

import (
	"context"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
)

type FixedStepResource struct {
	specDef *SpecDefinition
}

func NewFixedStepResource(specDef *SpecDefinition) *FixedStepResource {
	return &FixedStepResource{specDef: specDef}
}

func (sr *FixedStepResource) Fetch(_ context.Context, _ *expression.InterpolationContext) (*SpecDefinition, error) {
	return sr.specDef, nil
}
