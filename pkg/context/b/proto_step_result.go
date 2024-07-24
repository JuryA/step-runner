package b

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type ProtoStepResultBuilder struct {
	name       string
	outputSpec map[string]*proto.Spec_Content_Output
	output     map[string]*structpb.Value
	specDef    *proto.SpecDefinition
}

func ProtoStepResult() *ProtoStepResultBuilder {
	return &ProtoStepResultBuilder{
		name:       "my-step",
		outputSpec: map[string]*proto.Spec_Content_Output{},
		output:     map[string]*structpb.Value{},
	}
}

func (bldr *ProtoStepResultBuilder) WithName(name string) *ProtoStepResultBuilder {
	bldr.name = name
	return bldr
}

func (bldr *ProtoStepResultBuilder) WithOutputSpec(name string, spec *proto.Spec_Content_Output) *ProtoStepResultBuilder {
	bldr.outputSpec[name] = spec
	return bldr
}

func (bldr *ProtoStepResultBuilder) WithOutput(name string, value *structpb.Value) *ProtoStepResultBuilder {
	bldr.output[name] = value
	return bldr
}

func (bldr *ProtoStepResultBuilder) WithProtoSpecDef(specDef *proto.SpecDefinition) *ProtoStepResultBuilder {
	bldr.specDef = specDef
	return bldr
}

func (bldr *ProtoStepResultBuilder) Build() *proto.StepResult {
	return &proto.StepResult{
		Step:           ProtoStep().WithName(bldr.name).Build(),
		SpecDefinition: ProtoSpecDef().WithOutputSpec(bldr.outputSpec).Build(),
		Status:         proto.StepResult_success,
		Outputs:        bldr.output,
		Exports:        map[string]string{},
		Env:            map[string]string{},
		ExecResult:     &proto.StepResult_ExecResult{Command: []string{"go", "run", "."}, WorkDir: "", ExitCode: 0},
		SubStepResults: nil,
	}
}
