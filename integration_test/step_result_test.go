package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestStepResult(t *testing.T) {
	t.Run("steps and parameters are recorded both expanded and not expanded", func(t *testing.T) {
		yaml := `
spec: {}
---
env:
  PLEASE: ${{ env.REPLACE_ME }}
  NAME: subby
run:
  - name: greet_steppy
    step: ./steps/greeting
    inputs:
      name: ${{ env.NAME }}`

		result, _, err := testutil.StepRunner(t).WithEnvKeyVal("REPLACE_ME", "replaced").Run(yaml)

		require.NoError(t, err)

		// Top level step definition should be recorded but not expanded.
		require.Equal(t, "${{ env.REPLACE_ME }}", result.SpecDefinition.Definition.Env["PLEASE"])
		require.Equal(t, "${{ env.NAME }}", result.SpecDefinition.Definition.Steps[0].Inputs["name"].GetStringValue())

		// Sub-step invocation should be expanded and recorded.
		require.Equal(t, "subby", result.SubStepResults[0].Step.Inputs["name"].GetStringValue())

		// Exec definition should be recorded but not expanded.
		require.Equal(t, "${{ work_dir }}", result.SubStepResults[0].SpecDefinition.Definition.Env["HOME"])

		// Exec environment should be expanded and recorded.
		require.NotContains(t, "work_dir", result.SubStepResults[0].Env["HOME"])

		// Exec results should be recorded and expanded.
		require.NotContains(t, "work_dir", result.SubStepResults[0].ExecResult.WorkDir)
		require.Equal(t, "--name=subby", result.SubStepResults[0].ExecResult.Command[3])

		// Sub-steps environment should be expanded and recorded.
		require.Equal(t, "replaced", result.Env["PLEASE"])
	})
}
