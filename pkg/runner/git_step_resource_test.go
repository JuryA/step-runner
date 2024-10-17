package runner_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestGitStepResource_Interpolate(t *testing.T) {
	t.Run("interpolates URL", func(t *testing.T) {
		view := bldr.InterpolationCtx().WithEnvVar("STEP", "echo").Build()

		gitResource := runner.NewGitStepResource("https://gitlab.com/steps/${{env.STEP}}", "main", []string{}, "step.yml")
		newResource, err := gitResource.Interpolate(view)
		require.NoError(t, err)
		require.NotSame(t, gitResource, newResource)
		require.Equal(t, "https://gitlab.com/steps/echo@main:/step.yml", newResource.Describe())
	})
}
