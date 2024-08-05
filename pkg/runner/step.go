package runner

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Step interface {
	Run(ctx ctx.Context, stepsCtx *StepsContext, specDefinition *proto.SpecDefinition) (*proto.StepResult, error)
}
