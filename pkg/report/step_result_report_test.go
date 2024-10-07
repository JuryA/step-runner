package report

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestStepResultReport_Write(t *testing.T) {
	stepResultsFile := "step-results.json"
	t.Cleanup(func() { _ = os.Remove(stepResultsFile) })

	err := NewStepResultReport(stepResultsFile, FormatJSON).Write(bldr.StepResult().Build())
	require.NoError(t, err)
	require.FileExists(t, stepResultsFile)
}
