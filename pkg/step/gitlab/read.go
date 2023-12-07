package gitlab_step

import (
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"gitlab.com/gitlab-org/step-runner/proto"
)

const scriptStep = "gitlab.com/components/script@v1.0"
const scriptStepInputScript = "script"

func processStepSyntacticSugar(step *proto.ExtendedDefinition_Step) error {
	if step.Inputs == nil {
		step.Inputs = make(map[string]*structpb.Value)
	}

	switch v := step.StepOrScript.(type) {
	case *proto.ExtendedDefinition_Step_Step:

	case *proto.ExtendedDefinition_Step_Script:
		step.StepOrScript = &proto.ExtendedDefinition_Step_Step{
			Step: scriptStep,
		}
		step.Inputs[scriptStepInputScript] = structpb.NewStringValue(v.Script)
	}

	return nil
}

func processStepsSyntacticSugar(def *proto.ExtendedDefinition) error {
	for _, step := range def.Steps {
		err := processStepSyntacticSugar(step)
		if err != nil {
			return err
		}
	}
	return nil
}

func processSyntacticSugar(def *proto.ExtendedDefinition) error {
	switch def.Type {
	case proto.DefinitionType_steps:
		return processStepsSyntacticSugar(def)

	default:
		return nil
	}
}

func convertExtDefinition(def *proto.ExtendedDefinition) (*proto.Definition, error) {
	// Use serialization to ensure that what we are exporting (after reducing)
	// has the same structure as proto.Definition
	content, err := step.Marshal(def)
	if err != nil {
		return nil, err
	}

	var definition proto.Definition
	err = step.Unmarshal(content, &definition)
	return &definition, err
}

func Parse(content, dir string) (*proto.StepDefinition, error) {
	var (
		spec          proto.Spec
		extDefinition proto.ExtendedDefinition
	)

	if err := step.Unmarshal(content, &spec, &extDefinition); err != nil {
		return nil, err
	}

	if err := processSyntacticSugar(&extDefinition); err != nil {
		return nil, err
	}

	def, err := convertExtDefinition(&extDefinition)
	if err != nil {
		return nil, err
	}

	return &proto.StepDefinition{
		Spec:       &spec,
		Definition: def,
		Dir:        dir,
	}, nil
}

func Read(filename string) (*proto.StepDefinition, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return Parse(string(buf), filepath.Dir(filename))
}
