package runner

import (
	"context"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// StepResource knows how to load a Step
type StepResource interface {
	Fetch(context.Context, *expression.InterpolationContext) (*proto.SpecDefinition, error)
}
