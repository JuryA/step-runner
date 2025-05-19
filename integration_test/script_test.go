package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestCanRunAScript(t *testing.T) {
	stepYml := `
spec:
---
run:
  - name: my_script
    script: echo hi`

	_, logs, err := testutil.StepRunner(t).Run(stepYml)
	require.NoError(t, err)
	require.Contains(t, logs, "hi")
}
