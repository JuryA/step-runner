package run

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	stepResultsFileJSON = "step-results.json"
)

func TestRunCmd(t *testing.T) {
	t.Run("runs action", func(t *testing.T) {
		_ = os.Remove(stepResultsFileJSON)

		cmd := NewCmd()
		cmd.SetArgs([]string{
			"--write-steps-results",
			"./testdata/run_action",
			"--step-results-file",
			stepResultsFileJSON,
		})
		err := cmd.Execute()
		require.NoError(t, err)
	})
}
