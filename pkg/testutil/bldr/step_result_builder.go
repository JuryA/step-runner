package bldr

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepResultBuilder struct {
	specDef *proto.SpecDefinition
	status  proto.StepResult_Status
	outputs map[string]*structpb.Value
}

func StepResult() *StepResultBuilder {
	return &StepResultBuilder{
		specDef: ProtoSpecDef().Build(),
		status:  proto.StepResult_success,
		outputs: map[string]*structpb.Value{},
	}
}

func (bldr *StepResultBuilder) WithOutput(name string, value *structpb.Value) *StepResultBuilder {
	bldr.outputs[name] = value
	return bldr
}

func (bldr *StepResultBuilder) WithSpecDef(specDef *proto.SpecDefinition) *StepResultBuilder {
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
		SpecDefinition: bldr.specDef,
		Status:         bldr.status,
		Outputs:        bldr.outputs,
		Exports:        make(map[string]string),
		ExecResult:     &proto.StepResult_ExecResult{},
	}
}
