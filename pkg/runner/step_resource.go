package runner

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// StepResource knows how to load a Step
type StepResource interface {
	Describer
	Fetch(ctx.Context, *expression.InterpolationContext) (*proto.SpecDefinition, error)
}
