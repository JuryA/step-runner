package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestLazilyLoadedStep(t *testing.T) {
	t.Run("loads and executes step", func(t *testing.T) {
		specDef := buildSpecDef()
		resourceLoader := &FixedSpecDefCache{specDef: specDef}
		parser := &FixedStepParser{step: &FixedResultStep{stepResult: buildStepResult(specDef, proto.StepResult_success)}}
		stepRef := &proto.Step{
			Name:   "step-name",
			Step:   &proto.Step_Reference{Filename: "step.yml"},
			Env:    map[string]string{},
			Inputs: map[string]*structpb.Value{},
		}

		step := NewLazilyLoadedStep(buildGlobalCtx(), resourceLoader, parser, stepRef)
		stepResult, err := step.Run(context.Background(), buildStepsCtx(), specDef)

		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, stepResult.Status)
	})

	t.Run("errors when inputs are provided that are not defined", func(t *testing.T) {
		specDef := buildSpecDef()
		resourceLoader := &FixedSpecDefCache{specDef: specDef}
		parser := &FixedStepParser{step: &FixedResultStep{stepResult: buildStepResult(specDef, proto.StepResult_success)}}
		stepRef := &proto.Step{
			Name: "step-name",
			Step: &proto.Step_Reference{Filename: "step.yml"},
			Env:  map[string]string{},
			Inputs: map[string]*structpb.Value{
				"not.defined": structpb.NewStringValue("123456789"),
			},
		}

		step := NewLazilyLoadedStep(buildGlobalCtx(), resourceLoader, parser, stepRef)
		_, err := step.Run(context.Background(), buildStepsCtx(), specDef)

		require.Error(t, err)
		require.Equal(t, `failed to run lazily-evaluated step "step-name": failed to load: step does not accept input with name "not.defined"`, err.Error())
	})
}

type FixedSpecDefCache struct {
	specDef *proto.SpecDefinition
}

func (c *FixedSpecDefCache) Get(_ context.Context, _ string, _ *proto.Step_Reference) (*proto.SpecDefinition, error) {
	return c.specDef, nil
}

type FixedStepParser struct {
	step Step
}

func (c *FixedStepParser) Parse(_ *proto.SpecDefinition, _ *Params, _ StepReference) (Step, error) {
	return c.step, nil
}
