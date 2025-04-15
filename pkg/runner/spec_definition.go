package runner

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type SpecDefinition struct {
	spec       *proto.Spec
	definition *proto.Definition
	dir        string
}

func NewSpecDefinition(spec *proto.Spec, definition *proto.Definition, dir string) *SpecDefinition {
	return &SpecDefinition{
		spec:       spec,
		definition: definition,
		dir:        dir,
	}
}

func (sd *SpecDefinition) ToProto() *proto.SpecDefinition {
	return &proto.SpecDefinition{
		Spec:       sd.spec,
		Definition: sd.definition,
	}
}

func (sd *SpecDefinition) Dir() string {
	return sd.dir
}

func (sd *SpecDefinition) SpecInputs() map[string]*proto.Spec_Content_Input {
	return sd.spec.Spec.Inputs
}

func (sd *SpecDefinition) SpecOutputs() map[string]*proto.Spec_Content_Output {
	return sd.spec.Spec.Outputs
}

func (sd *SpecDefinition) IsTypeExec() bool {
	return sd.definition.Type == proto.DefinitionType_exec
}

func (sd *SpecDefinition) IsTypeSteps() bool {
	return sd.definition.Type == proto.DefinitionType_steps
}

func (sd *SpecDefinition) Steps() []*proto.Step {
	return sd.definition.Steps
}

func (sd *SpecDefinition) DescribeType() string {
	return sd.definition.Type.String()
}

func (sd *SpecDefinition) ExecCommand() []string {
	return sd.definition.Exec.Command
}

func (sd *SpecDefinition) Env() map[string]string {
	return sd.definition.Env
}

func (sd *SpecDefinition) ExecWorkDir() string {
	return sd.definition.Exec.WorkDir
}

func (sd *SpecDefinition) IsDelegateOutputs() bool {
	return sd.spec.Spec.OutputMethod == proto.OutputMethod_delegate
}

func (sd *SpecDefinition) DelegateTo() string {
	return sd.definition.Delegate
}

func (sd *SpecDefinition) DefinitionOutputs() map[string]*structpb.Value {
	return sd.definition.Outputs
}

func (sd *SpecDefinition) SpecInputWithName(name string) (*proto.Spec_Content_Input, bool) {
	input, ok := sd.spec.Spec.Inputs[name]
	return input, ok
}
