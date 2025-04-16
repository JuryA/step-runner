package runner_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestGitStepResource_Fetch(t *testing.T) {
	t.Run("loads Git step", func(t *testing.T) {
		repo, worktree := bldr.GitRepository().Build(t)
		gitServerURL := bldr.StartGitSmartHTTPServer(t, repo)

		bldr.GitWorktree(t, worktree).
			CreateFile("step.yml", "spec:\n---\nexec: {command: [bash]}").
			Stage("step.yml").
			Commit("Add step definition")

		view := bldr.InterpolationCtx().WithEnvVar("SERVER_ADDR", gitServerURL).Build()

		fetcher := git.New(t.TempDir(), git.CloneOptions{Depth: 0})
		res := runner.NewGitStepResource(fetcher, "${{env.SERVER_ADDR}}", "main", "", "step.yml")
		specDef, err := res.Fetch(context.Background(), view)
		require.NoError(t, err)
		require.Equal(t, []string{"bash"}, specDef.ExecCommand())
	})

	t.Run("loads Git step in sub-directory", func(t *testing.T) {
		repo, worktree := bldr.GitRepository().Build(t)
		gitServerURL := bldr.StartGitSmartHTTPServer(t, repo)

		commit := bldr.GitWorktree(t, worktree).
			MakeDir("foo/bar/bob").
			CreateFile("foo/bar/bob/step.yml", "spec:\n---\nexec: {command: [bash]}").
			Stage(".").
			Commit("Add step definition")

		fetcher := git.New(t.TempDir(), git.CloneOptions{Depth: 0})
		res := runner.NewGitStepResource(fetcher, gitServerURL, commit, "foo/bar/bob", "step.yml")
		specDef, err := res.Fetch(context.Background(), bldr.InterpolationCtx().Build())
		require.NoError(t, err)
		require.Equal(t, []string{"bash"}, specDef.ExecCommand())
	})
}
