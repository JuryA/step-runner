package e2e_tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestBasicStepFunctionality(t *testing.T) {
	t.Run("greeting with defaults", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: greet_steppy
    step: ./steps/greeting
    inputs: {}`

		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "steppy", result.SubStepResults[0].Outputs["name"].GetStringValue())
		require.Equal(t, "steppy", result.SubStepResults[0].Exports["NAME"])
	})

	t.Run("greeting outputs and exports name parameter", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: greet_foo
    step: ./steps/greeting
    inputs:
      name: foo`

		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "foo", result.SubStepResults[0].Outputs["name"].GetStringValue())
		require.Equal(t, "foo", result.SubStepResults[0].Exports["NAME"])
	})

	t.Run("return results even with an error", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: bang
    script: exit 1`

		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.Error(t, err)
		require.Contains(t, err.Error(), "step \"bang\": exec: exit status 1")
		require.NotNil(t, result)
		require.Equal(t, proto.StepResult_failure, result.Status)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, proto.StepResult_failure, result.SubStepResults[0].Status)
		require.Equal(t, int32(1), result.SubStepResults[0].ExecResult.ExitCode)
	})
}
