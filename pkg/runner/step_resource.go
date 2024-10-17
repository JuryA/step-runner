package runner

import (
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// StepResource knows how to load a Step
type StepResource interface {
	Describer
	Interpolate(*expression.InterpolationContext) (StepResource, error)
	ToProtoStepRef() *proto.Step_Reference
}
