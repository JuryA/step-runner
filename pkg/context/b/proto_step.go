package b

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type ProtoStepBuilder struct {
	name string
	env  map[string]string
}

func ProtoStep() *ProtoStepBuilder {
	return &ProtoStepBuilder{
		name: "my_step",
		env:  make(map[string]string),
	}
}

func (bldr *ProtoStepBuilder) WithName(name string) *ProtoStepBuilder {
	bldr.name = name
	return bldr
}

func (bldr *ProtoStepBuilder) WithEnvVar(name, value string) *ProtoStepBuilder {
	bldr.env[name] = value
	return bldr
}

func (bldr *ProtoStepBuilder) Build() *proto.Step {
	return &proto.Step{
		Name:   bldr.name,
		Step:   &proto.Step_Reference{Filename: "step.yml"},
		Env:    bldr.env,
		Inputs: map[string]*structpb.Value{},
	}
}
