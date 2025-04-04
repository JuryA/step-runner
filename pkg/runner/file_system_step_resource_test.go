package runner_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestFileSystemStepResource_Fetch(t *testing.T) {
	t.Run("loads local step", func(t *testing.T) {
		dir := bldr.Files(t).
			WriteFile("step.yml", "spec:\n---\nexec: {command: [sh]}").
			Build()

		resource := runner.NewFileSystemStepResource(dir, "step.yml")
		specDef, err := resource.Fetch(context.Background())
		require.NoError(t, err)
		require.Contains(t, specDef.Definition.Exec.Command, "sh")
	})
}
