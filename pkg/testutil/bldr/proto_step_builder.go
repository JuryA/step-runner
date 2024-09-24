package bldr

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type ProtoStepBuilder struct {
	name string
}

func ProtoStep() *ProtoStepBuilder {
	return &ProtoStepBuilder{
		name: "my-step",
	}
}

func (bldr *ProtoStepBuilder) WithName(name string) *ProtoStepBuilder {
	bldr.name = name
	return bldr
}

func (bldr *ProtoStepBuilder) Build() *proto.Step {
	return &proto.Step{
		Name: bldr.name,
		Step: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "gitlab.com/components/my-step",
			Path:     nil,
			Filename: "",
			Version:  "",
		},
		Env:    map[string]string{},
		Inputs: map[string]*structpb.Value{},
	}
}
