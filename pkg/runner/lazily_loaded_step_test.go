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
		specDef := bldr.ProtoSpecDef().Build()
		resourceLoader := &FixedSpecDefCache{specDef: specDef}

		stepResult := bldr.StepResult().WithSpecDef(specDef).WithSuccessStatus().Build()
		parser := &FixedStepParser{step: bldr.Step().WithRunReturnsStepResult(stepResult).Build()}

		stepRef := &proto.Step{
			Name:   "step-name",
			Step:   &proto.Step_Reference{Filename: "step.yml"},
			Env:    map[string]string{},
			Inputs: map[string]*structpb.Value{},
		}

		globalCtx := bldr.GlobalContext().Build()
		stepsCtx := bldr.StepsContext().Build()
		step := runner.NewLazilyLoadedStep(globalCtx, resourceLoader, parser, stepRef)
		stepResult, err := step.Run(context.Background(), stepsCtx, specDef)

		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, stepResult.Status)
	})

	t.Run("errors when inputs are provided that are not defined", func(t *testing.T) {
		specDef := bldr.ProtoSpecDef().Build()
		resourceLoader := &FixedSpecDefCache{specDef: specDef}

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
		stepsCtx := bldr.StepsContext().Build()
		step := runner.NewLazilyLoadedStep(globalCtx, resourceLoader, parser, stepRef)
		_, err := step.Run(context.Background(), stepsCtx, specDef)

		require.Error(t, err)
		require.Equal(t, `failed to run lazily-evaluated step "step-name": failed to load: step does not accept input with name "not.defined"`, err.Error())
	})

	t.Run("expands step reference URL", func(t *testing.T) {
		specDef := bldr.ProtoSpecDef().Build()
		resourceLoader := &FixedSpecDefCache{specDef: specDef}

		stepResult := bldr.StepResult().WithSpecDef(specDef).WithSuccessStatus().Build()
		parser := &FixedStepParser{step: bldr.Step().WithRunReturnsStepResult(stepResult).Build()}

		stepRef := &proto.Step{
			Name: "step-name",
			Step: &proto.Step_Reference{
				Url:      "http://gitlab-ci-token:${{ job.CI_JOB_TOKEN }}@gitlab.com/project.git",
				Filename: "step.yml",
			},
			Env:    map[string]string{},
			Inputs: map[string]*structpb.Value{},
		}

		globalCtx := bldr.GlobalContext().WithJob("CI_JOB_TOKEN", "ABCDEF").Build()
		stepsCtx := bldr.StepsContext().WithGlobalContext(globalCtx).Build()
		step := runner.NewLazilyLoadedStep(globalCtx, resourceLoader, parser, stepRef)
		stepResult, err := step.Run(context.Background(), stepsCtx, specDef)

		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, stepResult.Status)
		require.Equal(t, "http://gitlab-ci-token:ABCDEF@gitlab.com/project.git", resourceLoader.lastGet.Url)
	})
}

type FixedSpecDefCache struct {
	specDef *proto.SpecDefinition
	lastGet *proto.Step_Reference
}

func (c *FixedSpecDefCache) Get(_ context.Context, _ string, ref *proto.Step_Reference) (*proto.SpecDefinition, error) {
	c.lastGet = ref
	return c.specDef, nil
}

type FixedStepParser struct {
	step runner.Step
}

func (c *FixedStepParser) Parse(_ *proto.SpecDefinition, _ *runner.Params, _ runner.StepReference) (runner.Step, error) {
	return c.step, nil
}
