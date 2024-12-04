package action

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	step_runner "gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/steps/action/pkg/runner"
)

type Config struct {
	Action      string
	ActionImage string
	Inputs      map[string]string
}

func Run(_ context.Context, stepsCtx *step_runner.StepsContext) error {
	cfg, err := buildConfig(stepsCtx.Inputs)
	if err != nil {
		return err
	}

	stepResult, err := runner.Run(cfg.Action, cfg.ActionImage, cfg.Inputs)
	if err != nil {
		return fmt.Errorf("running action %q: %v", cfg.Action, err)
	}

	data, err := protojson.Marshal(stepResult)
	if err != nil {
		return fmt.Errorf("marshaling step result: %v", err)
	}

	err = os.WriteFile(stepsCtx.OutputFile.Path(), data, 0600)
	if err != nil {
		return fmt.Errorf("writing output file: %v", err)
	}

	return nil
}

func buildConfig(inputs map[string]*structpb.Value) (*Config, error) {
	if _, ok := inputs["action"]; !ok {
		return nil, fmt.Errorf("action is required")
	}

	if _, ok := inputs["actionImage"]; !ok {
		return nil, fmt.Errorf("actionImage is required")
	}

	actionInputs := make(map[string]string)

	// TODO: case where inputs is not provided
	// TODO: case where values are wrong types
	for name, value := range inputs["inputs"].GetStructValue().GetFields() {
		actionInputs[name] = value.GetStringValue()
	}

	cfg := &Config{
		Action:      inputs["action"].GetStringValue(),
		ActionImage: inputs["actionImage"].GetStringValue(),
		Inputs:      actionInputs,
	}

	return cfg, nil
}
