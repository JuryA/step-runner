package domain

import (
	ctx "context"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
)

type MultiStep struct{}

func NewMultiStep() *MultiStep {
	return &MultiStep{}
}

func (ms *MultiStep) Run(ctx.Context, *context.Global, *context.Steps) (*StepResult, error) {
	return nil, nil
}
