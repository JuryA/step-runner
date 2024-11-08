package runner_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestStepsContext_ExpandAndApplyEnv(t *testing.T) {
	globalCtx := bldr.GlobalContext().Build()

	inputs := map[string]*structpb.Value{"name": structpb.NewStringValue("sally")}
	env := runner.NewEnvironment(map[string]string{"HOME": "/home"})

	stepsCtx, err := runner.NewStepsContext(globalCtx, "", inputs, env)
	require.NoError(t, err)

	err = stepsCtx.ExpandAndApplyEnv(map[string]string{"WORK_DIR": "/home/${{ inputs.name }}"})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"HOME": "/home", "WORK_DIR": "/home/sally"}, stepsCtx.Env.Values())
}

func TestStepsContext_View(t *testing.T) {
	t.Run("can access step outputs", func(t *testing.T) {
		stepResults := map[string]*proto.StepResult{
			"step.a": bldr.StepResult().WithOutput("name", structpb.NewStringValue("step_a")).Build(),
			"step.b": bldr.StepResult().WithOutput("name", structpb.NewStringValue("step_b")).Build(),
		}

		stepsCtx := bldr.StepsContext(t).WithStepResults(stepResults).Build()
		view := stepsCtx.View()

		require.Len(t, view.StepResults, 2)
		require.Equal(t, "step_a", view.StepResults["step.a"].Outputs["name"].GetStringValue())
		require.Equal(t, "step_b", view.StepResults["step.b"].Outputs["name"].GetStringValue())
	})
}
