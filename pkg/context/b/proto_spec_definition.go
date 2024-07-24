package b

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type ProtoSpecDefinitionBuilder struct {
	outputSpec map[string]*proto.Spec_Content_Output
}

func ProtoSpecDef() *ProtoSpecDefinitionBuilder {
	return &ProtoSpecDefinitionBuilder{
		outputSpec: make(map[string]*proto.Spec_Content_Output),
	}
}

func (bldr *ProtoSpecDefinitionBuilder) WithOutputSpec(outputSpec map[string]*proto.Spec_Content_Output) *ProtoSpecDefinitionBuilder {
	bldr.outputSpec = outputSpec
	return bldr
}

func (bldr *ProtoSpecDefinitionBuilder) Build() *proto.SpecDefinition {
	return &proto.SpecDefinition{
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
	}
}
