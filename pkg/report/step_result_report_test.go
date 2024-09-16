package report

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestStepResultReport_Write(t *testing.T) {
	t.Cleanup(func() { _ = os.Remove(StepResultsFile) })

	err := NewStepResultReport().Write(bldr.StepResult().Build())
	require.NoError(t, err)
	require.FileExists(t, StepResultsFile)
}
