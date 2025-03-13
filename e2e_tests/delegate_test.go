package e2e_tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestDelegation(t *testing.T) {
	t.Run("delegate to exec step", func(t *testing.T) {
		yaml := `
spec:
  outputs: delegate
---
run:
  - name: exec_step
    step: ./test_steps/greeting
    inputs:
      name: steppy loves delegation
delegate: exec_step`

		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Equal(t, "steppy loves delegation", result.Outputs["name"].GetStringValue())
	})

	t.Run("delegate to composite step", func(t *testing.T) {
		yaml := `
spec:
  outputs: delegate
---
run:
  - name: composite_step
    step: ./test_steps/greeting_delegate
    inputs:
      name: steppy loves delegation
delegate: composite_step
`
		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Equal(t, "steppy loves delegation", result.Outputs["name"].GetStringValue())
	})
}
