package bldr

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepResultBuilder struct {
	specDef *runner.SpecDefinition
	status  proto.StepResult_Status
	step    *proto.Step
	outputs map[string]*structpb.Value
}

func StepResult() *StepResultBuilder {
	return &StepResultBuilder{
		specDef: SpecDef().Build(),
		status:  proto.StepResult_success,
		step:    nil,
		outputs: map[string]*structpb.Value{},
	}
}

func (bldr *StepResultBuilder) WithOutput(name string, value *structpb.Value) *StepResultBuilder {
	bldr.outputs[name] = value
	return bldr
}

func (bldr *StepResultBuilder) WithSpecDef(specDef *runner.SpecDefinition) *StepResultBuilder {
	bldr.specDef = specDef
	return bldr
}

func (bldr *StepResultBuilder) WithFailedStatus() *StepResultBuilder {
	bldr.status = proto.StepResult_failure
	return bldr
}

func (bldr *StepResultBuilder) WithSuccessStatus() *StepResultBuilder {
	bldr.status = proto.StepResult_success
	return bldr
}

func (bldr *StepResultBuilder) Build() *proto.StepResult {
	return &proto.StepResult{
		Step:           bldr.step,
		SpecDefinition: bldr.specDef.ToProto(),
		Status:         bldr.status,
		Outputs:        bldr.outputs,
		Exports:        make(map[string]string),
		Env:            nil,
		ExecResult:     &proto.StepResult_ExecResult{},
		SubStepResults: nil,
	}
}
