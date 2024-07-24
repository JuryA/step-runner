package b

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type ProtoStepResultBuilder struct {
	name       string
	outputSpec map[string]*proto.Spec_Content_Output
	output     map[string]*structpb.Value
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

func (bldr *ProtoStepResultBuilder) Build() *proto.StepResult {
	return &proto.StepResult{
		Step: &proto.Step{
			Name:   bldr.name,
			Step:   &proto.Step_Reference{Filename: "step.yml"},
			Env:    map[string]string{},
			Inputs: map[string]*structpb.Value{},
		},
		SpecDefinition: &proto.SpecDefinition{
			Spec: &proto.Spec{
				Spec: &proto.Spec_Content{
					Inputs:       map[string]*proto.Spec_Content_Input{},
					Outputs:      bldr.outputSpec,
					OutputMethod: proto.OutputMethod_outputs,
				},
			},
			Definition: &proto.Definition{
				Type:     proto.DefinitionType_exec,
				Exec:     &proto.Definition_Exec{Command: []string{"go", "run", "."}, WorkDir: ""},
				Steps:    nil,
				Outputs:  map[string]*structpb.Value(nil),
				Env:      map[string]string{},
				Delegate: "",
			},
			Dir: "",
		},
		Status:         proto.StepResult_success,
		Outputs:        bldr.output,
		Exports:        map[string]string{},
		Env:            map[string]string{},
		ExecResult:     &proto.StepResult_ExecResult{Command: []string{"go", "run", "."}, WorkDir: "", ExitCode: 0},
		SubStepResults: nil,
	}
}
