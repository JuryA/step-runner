package runner

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

// StepResult captures the result of running a Step
type StepResult struct {
	protoStepResult *proto.StepResult
}

func NewStepResult(protoStepResult *proto.StepResult) *StepResult {
	return &StepResult{
		protoStepResult: protoStepResult,
	}
}

func (sr *StepResult) StepName() (bool, string) {
	if sr.protoStepResult.Step == nil {
		return false, ""
	}

	return true, sr.protoStepResult.Step.Name
}

func (sr *StepResult) ProtoStepResult() *proto.StepResult {
	return sr.protoStepResult
}

func (sr *StepResult) Outputs() map[string]*structpb.Value {
	return sr.protoStepResult.Outputs
}

func (sr *StepResult) Failed() bool {
	return sr.protoStepResult.Status == proto.StepResult_failure
}
