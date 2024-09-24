package runner

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Step interface {
	Describer
	Run(ctx ctx.Context, stepsCtx *StepsContext, specDefinition *proto.SpecDefinition) (*StepResult, error)
}
