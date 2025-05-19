package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestEnvironmentVariablesExpansion(t *testing.T) {
	t.Run("steps environment variables are expanded", func(t *testing.T) {
		yaml := `
spec: {}
---
env:
  NAME: from-${{ env.WHERE_EXACTLY }}
run:
  - name: greet_steppy
    step: ./steps/greeting_name_from_env`

		result, _, err := testutil.StepRunner(t).WithEnvKeyVal("WHERE_EXACTLY", "global").Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "from-global", result.SubStepResults[0].Outputs["name"].GetStringValue())
	})

	t.Run("step invocation environment variables are expanded", func(t *testing.T) {
		yaml := `
spec: {}
---
env:
run:
  - name: greet_steppy
    step: ./steps/greeting_name_from_env
    env:
      NAME: from-${{ env.WHERE_EXACTLY }}`

		result, _, err := testutil.StepRunner(t).WithEnvKeyVal("WHERE_EXACTLY", "global").Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "from-global", result.SubStepResults[0].Outputs["name"].GetStringValue())
	})

	t.Run("steps environment variables are expanded before invocation", func(t *testing.T) {
		yaml := `
spec: {}
---
env:
  WHERE_EXACTLY: ${{ env.WHERE_EXACTLY }}-then-run
run:
  - name: greet_steppy
    step: ./steps/greeting_name_from_env
    env:
      NAME: from-${{ env.WHERE_EXACTLY }}-then-invocation`

		result, _, err := testutil.StepRunner(t).WithEnvKeyVal("WHERE_EXACTLY", "global").Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "from-global-then-run-then-invocation", result.SubStepResults[0].Outputs["name"].GetStringValue())
	})
}
