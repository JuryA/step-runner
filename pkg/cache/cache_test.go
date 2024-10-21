package cache_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestCache(t *testing.T) {
	t.Run("loads local step", func(t *testing.T) {
		stepCache, err := cache.New()
		require.NoError(t, err)

		res := bldr.FileSystemStepResource().Build()
		specDef, err := stepCache.Get(context.Background(), "../runner/test_steps/echo", res)
		require.NoError(t, err)
		require.Contains(t, strings.Join(specDef.Definition.Exec.Command, ","), "echo")
	})

	t.Run("loads Git step", func(t *testing.T) {
		gitFetcher := git.New(t.TempDir(), git.CloneOptions{Depth: 0})
		stepCache := cache.NewWithOptions(gitFetcher)
		repo, worktree := bldr.GitRepository().Build(t)
		gitServerURL := bldr.StartGitSmartHTTPServer(t, repo)

		bldr.GitWorktree(t, worktree).
			CreateFile("step.yml", "spec:\n---\nexec: {command: [bash]}").
			Stage("step.yml").
			Commit("Add step definition")

		res := bldr.GitStepResource().WithURL(gitServerURL).WithVersion("main").Build()
		specDef, err := stepCache.Get(context.Background(), t.TempDir(), res)
		require.NoError(t, err)
		require.Equal(t, []string{"bash"}, specDef.Definition.Exec.Command)
	})

	t.Run("loads Git step in sub-directory", func(t *testing.T) {
		gitFetcher := git.New(t.TempDir(), git.CloneOptions{Depth: 0})
		stepCache := cache.NewWithOptions(gitFetcher)
		repo, worktree := bldr.GitRepository().Build(t)
		gitServerURL := bldr.StartGitSmartHTTPServer(t, repo)

		commit := bldr.GitWorktree(t, worktree).
			MakeDir("foo/bar/bob").
			CreateFile("foo/bar/bob/step.yml", "spec:\n---\nexec: {command: [bash]}").
			Stage(".").
			Commit("Add step definition")

		res := bldr.GitStepResource().
			WithURL(gitServerURL).
			WithPath("foo", "bar", "bob").
			WithVersion(commit).
			Build()
		specDef, err := stepCache.Get(context.Background(), t.TempDir(), res)
		require.NoError(t, err)
		require.Equal(t, []string{"bash"}, specDef.Definition.Exec.Command)
	})
}
