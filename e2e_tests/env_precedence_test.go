package e2e_tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestEnvironmentVariablePrecedence(t *testing.T) {
	t.Run("steps environment takes precedence over global environment", func(t *testing.T) {
		yaml := `
spec: {}
---
env:
  NAME: from-run
run:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs:
      name: ${{ env.NAME }}`

		result, _, err := testutil.StepRunner(t).WithEnvKeyVal("NAME", "from-global").Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "from-run", result.SubStepResults[0].Outputs["name"].GetStringValue())
	})

	t.Run("individual step invocation environment takes precedence over global environment", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: greet_steppy
    step: ./test_steps/greeting_name_from_env
    env:
      NAME: from-step-invocation`

		result, _, err := testutil.StepRunner(t).WithEnvKeyVal("NAME", "from-global").Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "from-step-invocation", result.SubStepResults[0].Outputs["name"].GetStringValue())
	})

	t.Run("individual step invocation environment takes precedence over steps environment", func(t *testing.T) {
		yaml := `
spec: {}
---
env:
  NAME: from-run
run:
  - name: greet_steppy
    step: ./test_steps/greeting_name_from_env
    env:
      NAME: from-step-invocation`

		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "from-step-invocation", result.SubStepResults[0].Outputs["name"].GetStringValue())
	})
}
