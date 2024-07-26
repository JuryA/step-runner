package domain

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
)

type Step interface {
	Run(ctx.Context, *context.Global, *context.Steps) (*StepResult, error)
}
