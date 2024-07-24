package context

import "gitlab.com/gitlab-org/step-runner/proto"

type Step struct {
	ProtoDef  *proto.SpecDefinition
	ProtoStep *proto.Step
	Inputs    map[string]*Variable
}

func NewStep(protoStep *proto.Step, protoDef *proto.SpecDefinition, inputs map[string]*Variable) *Step {
	if protoStep == nil {
		panic("proto step cannot be nil")
	}

	if protoDef == nil {
		panic("proto definition cannot be nil")
	}

	return &Step{
		ProtoDef:  protoDef,
		ProtoStep: protoStep,
		Inputs:    inputs,
	}
}

func (s *Step) Name() string {
	return s.ProtoStep.Name
}

func (s *Step) Env() map[string]string {
	return s.ProtoStep.Env
}
