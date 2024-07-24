package context

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

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

func (s *Step) ExpandInputs(stepsCtx *Steps, expand func(obj any, value *structpb.Value) (*Value, error)) (map[string]*Variable, error) {
	expanded := make(map[string]*Variable)

	for name, input := range s.Inputs {
		value, err := expand(stepsCtx, input.Value)

		if err != nil {
			return nil, fmt.Errorf("failed to expand input %q: %w", name, err)
		}

		expanded[name], err = s.Inputs[name].Assign(value)

		if err != nil {
			return nil, fmt.Errorf("failed to expand input %q: %w", name, err)
		}
	}

	return expanded, nil
}
