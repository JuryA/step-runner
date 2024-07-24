package b

import (
	"gitlab.com/gitlab-org/step-runner/proto"
)

type ProtoSpecDefinitionBuilder struct {
}

func ProtoSpecDef() *ProtoSpecDefinitionBuilder {
	return &ProtoSpecDefinitionBuilder{}
}

func (bldr *ProtoSpecDefinitionBuilder) Build() *proto.SpecDefinition {
	return &proto.SpecDefinition{
		Spec: &proto.Spec{
			Spec: &proto.Spec_Content{
				Inputs:       map[string]*proto.Spec_Content_Input{},
				Outputs:      map[string]*proto.Spec_Content_Output{},
				OutputMethod: proto.OutputMethod_outputs,
			},
		},
	}
}
