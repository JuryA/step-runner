package domain

import (
	ctx "context"
)

type Step interface {
	Run(ctx.Context, *GlobalCtx, *StepsCtx) (StepResult, error)
}
