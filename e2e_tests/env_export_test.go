package e2e_tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestEnvironmentExport(t *testing.T) {
	t.Run("exported env can be used in subsequent step", func(t *testing.T) {
		yaml := `
spec:
---
run:
  - name: set_export_var
    step: ./test_steps/export_env
    inputs:
      name: FOO
      value: BAR
  - name: verify_foo_can_be_used
    step: ./test_steps/echo
    inputs:
      echo: "FOO is ${{env.FOO}}"
`
		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 2)
		require.Equal(t, "FOO is BAR", result.SubStepResults[1].Outputs["echo"].GetStringValue())
	})
}
