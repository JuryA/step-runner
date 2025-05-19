package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestRunStepUsingDynamicStepRef(t *testing.T) {
	yaml := `
spec: {}
---
run:
  - name: env_step
    step: ./steps/export_env
    inputs:
      name: STEP_REF
      value: ./steps/greeting
  - name: next_step
    step: ${{env.STEP_REF}}`

	result, _, err := testutil.StepRunner(t).Run(yaml)
	require.NoError(t, err)
	require.Len(t, result.SubStepResults, 2)
	require.Equal(t, "steppy", result.SubStepResults[1].Outputs["name"].GetStringValue())
}
