package runner_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestLazilyLoadedStep(t *testing.T) {
	t.Run("loads and executes step", func(t *testing.T) {
		specDef := bldr.SpecDef().Build()

		stepResult := bldr.StepResult().WithSpecDef(specDef).WithSuccessStatus().Build()
		parser := &FixedStepParser{step: bldr.Step().WithRunReturnsStepResult(stepResult).Build()}

		stepRef := &proto.Step{
			Name:   "step-name",
			Step:   &proto.Step_Reference{Filename: "step.yml"},
			Env:    map[string]string{},
			Inputs: map[string]*structpb.Value{},
		}

		globalCtx := bldr.GlobalContext().Build()
		stepsCtx := bldr.StepsContext(t).Build()
		stepResource := bldr.StepResource(specDef).Build()
		step := runner.NewLazilyLoadedStep(globalCtx, parser, stepRef, stepResource)
		stepResult, err := step.Run(context.Background(), stepsCtx)

		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, stepResult.Status)
	})

	t.Run("errors when inputs are provided that are not defined", func(t *testing.T) {
		specDef := bldr.SpecDef().Build()

		stepResult := bldr.StepResult().WithSpecDef(specDef).WithSuccessStatus().Build()
		parser := &FixedStepParser{step: bldr.Step().WithRunReturnsStepResult(stepResult).Build()}
		stepRef := &proto.Step{
			Name: "step-name",
			Step: &proto.Step_Reference{Filename: "step.yml"},
			Env:  map[string]string{},
			Inputs: map[string]*structpb.Value{
				"not.defined": structpb.NewStringValue("123456789"),
			},
		}

		globalCtx := bldr.GlobalContext().Build()
		stepsCtx := bldr.StepsContext(t).Build()
		stepResource := bldr.StepResource(specDef).Build()
		step := runner.NewLazilyLoadedStep(globalCtx, parser, stepRef, stepResource)
		_, err := step.Run(context.Background(), stepsCtx)

		require.Error(t, err)
		require.Equal(t, `step "step-name": failed to load: step does not accept input with name "not.defined"`, err.Error())
	})
}

type FixedSpecDefCache struct {
	specDef *proto.SpecDefinition
	lastGet runner.StepResource
}

func (c *FixedSpecDefCache) Get(_ context.Context, stepResource runner.StepResource) (*proto.SpecDefinition, error) {
	c.lastGet = stepResource
	return c.specDef, nil
}

type FixedStepParser struct {
	step runner.Step
}

func (c *FixedStepParser) Parse(_ *runner.GlobalContext, _ *runner.SpecDefinition, _ *runner.Params, _ runner.StepReference) (runner.Step, error) {
	return c.step, nil
}
