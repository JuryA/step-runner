package runner_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestStepResults_FindOutputsForStepName(t *testing.T) {
	t.Run("finds outputs for name", func(t *testing.T) {
		stepResult := bldr.StepResult().
			WithStep(bldr.ProtoStep().WithName("my.step").Build()).
			WithOutput("key", structpb.NewStringValue("value")).
			Build()

		outputs, err := runner.NewStepResults(stepResult).FindOutputsForStepName("my.step")
		require.NoError(t, err)
		require.Equal(t, "value", outputs["key"].GetStringValue())
	})

	t.Run("returns error when name not found", func(t *testing.T) {
		stepResult := bldr.StepResult().Build()

		_, err := runner.NewStepResults(stepResult).FindOutputsForStepName("non.existent.step")
		require.Error(t, err)
		require.Equal(t, `delegating outputs to "non.existent.step": could not find substep`, err.Error())
	})
}
