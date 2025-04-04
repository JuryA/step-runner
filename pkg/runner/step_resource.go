package runner

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// StepResource knows how to load a Step
type StepResource interface {
	Describer
	Interpolate(*expression.InterpolationContext) (StepResource, error)
	Fetch(ctx ctx.Context) (*proto.SpecDefinition, error)
}
