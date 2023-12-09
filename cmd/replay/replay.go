package replay

import (
	ctx "context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/testing/protocmp"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

var Cmd = &cobra.Command{
	Use:   "replay <step-results.json>",
	Short: "Rerun step results",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("reading step results file %v: %w", args[0], err)
	}
	result := &proto.StepResult{}
	err = json.Unmarshal(data, result)
	if err != nil {
		return fmt.Errorf("unmarshaling step results: %w", err)
	}
	defs, err := cache.New()
	if err != nil {
		return fmt.Errorf("creating cache: %w", err)
	}
	globalCtx := context.NewGlobal()
	globalCtx.InheritEnv(os.Environ()...)
	execution, err := runner.New(defs)
	if err != nil {
		return fmt.Errorf("creating execution: %w", err)
	}
	stepDef, params, err := getStepDefinitionAndParams(result)
	if err != nil {
		return err
	}
	replay, err := execution.Run(ctx.Background(), stepDef, params, globalCtx)
	if err != nil {
		return fmt.Errorf("running execution: %w", err)
	}
	if diff := cmp.Diff(result, replay, protocmp.Transform()); diff != "" {
		return fmt.Errorf("replay was not the same as original result:\n%v", diff)
	}
	return nil
}

func getStepDefinitionAndParams(result *proto.StepResult) (*proto.StepDefinition, *runner.Params, error) {
	steps, err := getSteps(result)
	if err != nil {
		return nil, nil, fmt.Errorf("getting steps: %w", err)
	}
	replayOutputs, err := getReplayOutputs(result)
	if err != nil {
		return nil, nil, fmt.Errorf("getting replay outputs: %w", err)
	}
	stepDef := &proto.StepDefinition{
		Spec: &proto.Spec{
			Spec: &proto.Spec_Content{},
		},
		Definition: &proto.Definition{
			Type:  proto.DefinitionType_steps,
			Steps: steps,
		},
	}
	params := &runner.Params{
		ReplayOutputs: replayOutputs,
	}
	return stepDef, params, nil
}

func getSteps(result *proto.StepResult) ([]*proto.Step, error) {
	if result == nil {
		return nil, fmt.Errorf("nil result")
	}
	if result.StepDefinition == nil {
		return nil, fmt.Errorf("nil step definition")
	}
	if result.StepDefinition.Definition == nil {
		return nil, fmt.Errorf("nil definition")
	}
	if len(result.StepDefinition.Definition.Steps) == 0 {
		return nil, fmt.Errorf("no steps")
	}
	return result.StepDefinition.Definition.Steps, nil
}

func getReplayOutputs(result *proto.StepResult) (*runner.ReplayOutputs, error) {
	replayOutputs := runner.NewReplayOutputs()
	for k, v := range result.Outputs {
		replayOutputs.Outputs[k] = v
	}
	for i, c := range result.ChildrenStepResults {
		if c.Step == nil {
			return nil, fmt.Errorf("no step in result %v", i)
		}
		name := c.Step.Name
		o, err := getReplayOutputs(c)
		if err != nil {
			return nil, fmt.Errorf("in %v: %w", name, err)
		}
		replayOutputs.ChildrenOutputs[name] = o
	}
	return replayOutputs, nil
}
