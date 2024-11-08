package bldr

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepBuilder struct {
	err        error
	stepResult *proto.StepResult
}

func Step() *StepBuilder {
	return &StepBuilder{
		stepResult: StepResult().Build(),
		err:        nil,
	}
}

func (bldr *StepBuilder) WithRunReturnsStepResult(stepResult *proto.StepResult) *StepBuilder {
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
	stepResult *proto.StepResult
	err        error
}

func (s *FixedResultStep) Describe() string {
	return fmt.Sprintf("fixed result step %s", s.stepResult.Status)
}

func (s *FixedResultStep) Run(_ context.Context, _ *runner.GlobalContext, _ string, _ map[string]*structpb.Value, _ *runner.Environment, _ map[string]*proto.StepResult) (*proto.StepResult, error) {
	return s.stepResult, s.err
}
