package context

import "gitlab.com/gitlab-org/step-runner/proto"

type Step struct {
	ProtoDef  *proto.SpecDefinition
	ProtoStep *proto.Step
}

func NewStep(protoStep *proto.Step, protoDef *proto.SpecDefinition) *Step {
	if protoStep == nil {
		panic("proto step cannot be nil")
	}

	if protoDef == nil {
		panic("proto definition cannot be nil")
	}

	return &Step{
		ProtoDef:  protoDef,
		ProtoStep: protoStep,
	}
}
