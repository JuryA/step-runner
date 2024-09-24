package bldr

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepBuilder struct {
	err        error
	stepResult *runner.StepResult
}

func Step() *StepBuilder {
	return &StepBuilder{
		stepResult: StepResult().Build(),
		err:        nil,
	}
}

func (bldr *StepBuilder) WithRunReturnsStepResult(stepResult *runner.StepResult) *StepBuilder {
	bldr.stepResult = stepResult
	return bldr
}

func (bldr *StepBuilder) WithRunReturnsErr(err error) *StepBuilder {
	bldr.err = err
	return bldr
}

func (bldr *StepBuilder) Build() *FixedResultStep {
	return &FixedResultStep{
		stepResult: bldr.stepResult,
		err:        bldr.err,
	}
}

type FixedResultStep struct {
	stepResult *runner.StepResult
	err        error
}

func (s *FixedResultStep) Describe() string {
	return fmt.Sprintf("fixed result step")
}

func (s *FixedResultStep) Run(_ context.Context, _ *runner.StepsContext, _ *proto.SpecDefinition) (*runner.StepResult, error) {
	return s.stepResult, s.err
}
