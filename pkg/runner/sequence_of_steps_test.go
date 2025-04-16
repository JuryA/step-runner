package runner_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestSequenceOfSteps_Describe(t *testing.T) {
	subStepA := bldr.Step().Build()
	subStepB := bldr.Step().Build()
	specDef := bldr.SpecDef().Build()

	steps := runner.NewSequenceOfSteps(runner.StepDefinedInGitLabJob, &runner.Params{}, specDef, subStepA, subStepB)
	require.Equal(t, "sequence of 2 steps", steps.Describe())
}

func TestSequenceOfSteps_Run(t *testing.T) {
	t.Run("sub-step succeeds", func(t *testing.T) {
		stepResult := bldr.StepResult().WithSuccessStatus().Build()
		subStep := bldr.Step().WithRunReturnsStepResult(stepResult).Build()
		stepsCtx := bldr.StepsContext(t).Build()
		specDef := bldr.SpecDef().Build()

		steps := runner.NewSequenceOfSteps(runner.StepDefinedInGitLabJob, &runner.Params{}, specDef, subStep)
		result, err := steps.Run(context.Background(), stepsCtx)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, proto.StepResult_success, result.Status)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, proto.StepResult_success, result.SubStepResults[0].Status)
	})

	t.Run("sub-step fails", func(t *testing.T) {
		err := fmt.Errorf("simulated.error")
		stepResult := bldr.StepResult().WithFailedStatus().Build()
		subStep := bldr.Step().WithRunReturnsStepResult(stepResult).WithRunReturnsErr(err).Build()
		stepsCtx := bldr.StepsContext(t).Build()
		specDef := bldr.SpecDef().Build()

		steps := runner.NewSequenceOfSteps(runner.StepDefinedInGitLabJob, &runner.Params{}, specDef, subStep)
		result, err := steps.Run(context.Background(), stepsCtx)
		require.Error(t, err)
		require.Equal(t, "simulated.error", err.Error())
		require.NotNil(t, result)
		require.Equal(t, proto.StepResult_failure, result.Status)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, proto.StepResult_failure, result.SubStepResults[0].Status)
	})

	t.Run("interpolates outputs", func(t *testing.T) {
		subStep := bldr.Step().Build()
		stepsCtx := bldr.StepsContext(t).WithEnv("FOO", "BAR").Build()

		protoDef := bldr.ProtoDef().
			WithOutput("name", structpb.NewStringValue("name is ${{env.FOO}}")).
			Build()
		specDef := bldr.SpecDef().WithDefinition(protoDef).Build()

		steps := runner.NewSequenceOfSteps(runner.StepDefinedInGitLabJob, &runner.Params{}, specDef, subStep)
		result, err := steps.Run(context.Background(), stepsCtx)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "name is BAR", result.Outputs["name"].GetStringValue())
	})
}
