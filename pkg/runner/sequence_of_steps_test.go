package runner_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestSequenceOfSteps_Describe(t *testing.T) {
	subStepA := bldr.Step().Build()
	subStepB := bldr.Step().Build()

	steps := runner.NewSequenceOfSteps(runner.StepDefinedInGitLabJob, &runner.Params{}, subStepA, subStepB)
	require.Equal(t, "sequence of 2 steps", steps.Describe())
}

func TestSequenceOfSteps_Run(t *testing.T) {
	t.Run("sub-step succeeds", func(t *testing.T) {
		stepResult := bldr.StepResult().WithSuccessStatus().Build()
		subStep := bldr.Step().WithRunReturnsStepResult(stepResult).Build()
		stepsCtx := bldr.StepsContext().Build()
		specDef := bldr.ProtoSpecDef().Build()

		steps := runner.NewSequenceOfSteps(runner.StepDefinedInGitLabJob, &runner.Params{}, subStep)
		result, err := steps.Run(context.Background(), stepsCtx, specDef)
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
		stepsCtx := bldr.StepsContext().Build()
		specDef := bldr.ProtoSpecDef().Build()

		steps := runner.NewSequenceOfSteps(runner.StepDefinedInGitLabJob, &runner.Params{}, subStep)
		result, err := steps.Run(context.Background(), stepsCtx, specDef)
		require.Error(t, err)
		require.Equal(t, "failed to run sequence of steps: simulated.error", err.Error())
		require.NotNil(t, result)
		require.Equal(t, proto.StepResult_failure, result.Status)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, proto.StepResult_failure, result.SubStepResults[0].Status)
	})
}
