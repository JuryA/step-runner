package runner

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Step interface {
	Run(ctx ctx.Context, stepsCtx *context.Steps, specDefinition *proto.SpecDefinition, result *proto.StepResult) error
}
