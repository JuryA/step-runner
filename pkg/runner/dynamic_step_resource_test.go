package runner_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/di"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestDynamicStepResource_Fetch(t *testing.T) {
	t.Run("fetches resource", func(t *testing.T) {
		dir := bldr.Files(t).WriteFile("step.yml", []byte("spec:\n---\nexec: {command: [sh]}")).Build()
		view := bldr.InterpolationCtx().
			WithEnvVar("MY_VAR_A", "${{env.MY_VAR_B}}").
			WithEnvVar("MY_VAR_B", dir).
			Build()

		container := di.NewContainer()
		stepRscParser, err := container.StepResourceParser()
		require.NoError(t, err)

		dynResource := runner.NewDynamicStepResource(stepRscParser, "${{env.MY_VAR_A}}")
		specDef, err := dynResource.Fetch(context.Background(), view)
		require.NoError(t, err)
		require.Contains(t, specDef.Definition.Exec.Command, "sh")
	})
}
