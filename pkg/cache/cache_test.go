package cache_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	stepdist "gitlab.com/gitlab-org/step-runner/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestCache(t *testing.T) {
	t.Run("loads local step", func(t *testing.T) {
		stepCache := cache.NewWithOptions(
			cache.WithGitFetcher(git.New(t.TempDir(), git.CloneOptions{Depth: 1})),
			cache.WithOCIFetcher(oci.NewOCIFetcher(t.TempDir())),
			cache.WithDistFetcher(dist.NewFetcher(stepdist.FindDistributedStep)))

		res := bldr.FileSystemStepResource(t).WithDir("../../e2e_tests/steps/echo").Build()
		specDef, err := stepCache.Get(context.Background(), res)
		require.NoError(t, err)
		require.Contains(t, strings.Join(specDef.Definition.Exec.Command, ","), "echo")
	})
}
