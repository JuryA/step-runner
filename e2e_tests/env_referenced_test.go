package e2e_tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestEnvironmentVariablesReferenced(t *testing.T) {
	t.Run("global environment can be referenced", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: greet_steppy
    step: ./steps/greeting
    inputs:
      name: ${{ env.NAME }}
`
		result, _, err := testutil.StepRunner(t).WithEnvKeyVal("NAME", "from-global").Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "from-global", result.SubStepResults[0].Outputs["name"].GetStringValue())
	})

	t.Run("run environment can be referenced", func(t *testing.T) {
		yaml := `
spec: {}
---
env:
  NAME: from-run
run:
  - name: greet_steppy
    step: ./steps/greeting
    inputs:
      name: ${{ env.NAME }}`

		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "from-run", result.SubStepResults[0].Outputs["name"].GetStringValue())
	})

	t.Run("individual step invocation environment cannot be referenced during invocation", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: greet_steppy
    step: ./steps/greeting
    env:
      NAME: from-step-invocation
    inputs:
      name: ${{ env.NAME }}`

		_, _, err := testutil.StepRunner(t).Run(yaml)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "greet_steppy": failed to load: expand input "name": env.NAME: the "NAME" was not found`)
	})

	t.Run("individual step invocation environment can be referenced by step", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: greet_steppy
    step: ./steps/greeting_name_from_env
    env:
      NAME: from-step-invocation
`
		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "from-step-invocation", result.SubStepResults[0].Outputs["name"].GetStringValue())
	})
}
