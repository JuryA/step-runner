package domain

import (
	goctx "context"
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
)

type MultiStep struct {
	steps []Step
}

func NewMultiStep(steps ...Step) *MultiStep {
	return &MultiStep{
		steps: steps,
	}
}

func (ms *MultiStep) Run(ctx goctx.Context, globalCtx *context.Global, stepCtx *context.Steps) (StepResult, error) {
	for _, step := range ms.steps {
		_, err := step.Run(ctx, globalCtx, stepCtx)

		if err != nil {
			return nil, fmt.Errorf("failed to run steps: %w", err)
		}
	}

	return nil, nil
}
