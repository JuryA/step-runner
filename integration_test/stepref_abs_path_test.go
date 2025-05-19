package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestRunStepLoadingFromAbsolutePath(t *testing.T) {
	absolutePath, err := filepath.Abs("./steps/greeting")
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(absolutePath))

	yaml := `
spec: {}
---
run:
  - name: next_step
    step: ` + absolutePath

	result, _, err := testutil.StepRunner(t).Run(yaml)
	require.NoError(t, err)
	require.Len(t, result.SubStepResults, 1)
	require.Equal(t, "steppy", result.SubStepResults[0].Outputs["name"].GetStringValue())
}
