package runner_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestFileSystemStepResource_Fetch(t *testing.T) {
	t.Run("load step using step path relative to work dir", func(t *testing.T) {
		workDir := bldr.Files(t).
			WriteFile("/path/to/step/step.yml", "spec:\n---\nexec: {command: [sh]}").
			Build()

		view := bldr.InterpolationCtx().WithEnvVar("STEP_PATH", "path/to/step").Build()
		resource := runner.NewFileSystemStepResource(workDir, "${{env.STEP_PATH}}", "step.yml")
		specDef, err := resource.Fetch(context.Background(), view)
		require.NoError(t, err)
		require.Contains(t, specDef.ToProto().Definition.Exec.Command, "sh")
	})

	t.Run("loads step using absolute step path", func(t *testing.T) {
		stepPath := bldr.Files(t).
			WriteFile("step.yml", "spec:\n---\nexec: {command: [sh]}").
			Build()

		view := bldr.InterpolationCtx().WithEnvVar("STEP_PATH", stepPath).Build()
		resource := runner.NewFileSystemStepResource(t.TempDir(), "${{env.STEP_PATH}}", "step.yml")
		specDef, err := resource.Fetch(context.Background(), view)
		require.NoError(t, err)
		require.Contains(t, specDef.ToProto().Definition.Exec.Command, "sh")
	})
}
