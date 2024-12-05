package action

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

var SpecDef = &proto.SpecDefinition{
	Spec: &proto.Spec{
		Spec: &proto.Spec_Content{
			Inputs: map[string]*proto.Spec_Content_Input{
				"action": {
					Type: proto.ValueType_string,
				},
				"actionImage": {
					Type:    proto.ValueType_string,
					Default: structpb.NewStringValue("catthehacker/ubuntu:act-latest"),
				},
				"inputs": {
					Type: proto.ValueType_struct,
				},
			},
			OutputMethod: proto.OutputMethod_delegate,
		},
	},
	Definition: &proto.Definition{
		Type: proto.DefinitionType_exec,
		Exec: &proto.Definition_Exec{
			Command: []string{
				"${{inputs.step_runner}}",
				"act",
				"--steps-context=${{steps_context}}",
			},
		},
	},
}
