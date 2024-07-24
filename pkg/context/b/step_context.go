package b

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepContextBuilder struct {
	stepResults map[string]*proto.StepResult
}

func StepContext() *StepContextBuilder {
	return &StepContextBuilder{
		stepResults: map[string]*proto.StepResult{},
	}
}

func (bldr *StepContextBuilder) WithStepResult(stepResult *proto.StepResult) *StepContextBuilder {
	bldr.stepResults[stepResult.Step.Name] = stepResult
	return bldr
}

func (bldr *StepContextBuilder) Build() *context.Steps {
	return &context.Steps{
		Global:     GlobalContext().Build(),
		StepDir:    ".",
		OutputFile: "output",
		Env:        map[string]string{},
		Inputs:     map[string]*structpb.Value{},
		Steps:      bldr.stepResults,
	}
}
