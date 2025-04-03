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
		stepCache, err := cache.New()
		require.NoError(t, err)

		res := bldr.FileSystemStepResource(t).WithDir("../../e2e_tests/steps/echo").Build()
		specDef, err := stepCache.Get(context.Background(), "", res)
		require.NoError(t, err)
		require.Contains(t, strings.Join(specDef.Definition.Exec.Command, ","), "echo")
	})

	t.Run("loads Git step", func(t *testing.T) {
		gitFetcher := git.New(t.TempDir(), git.CloneOptions{Depth: 0})
		stepCache := cache.NewWithOptions(cache.WithGitFetcher(gitFetcher))
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
		stepCache := cache.NewWithOptions(cache.WithGitFetcher(gitFetcher))
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

	t.Run("loads OCI step", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		layer := bldr.OCIImageLayer(t).WithFile("/step.yml", []byte("spec:\n---\nexec: {command: [bash]}")).Build()
		img := bldr.OCIImage(t).WithLayer(layer).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(remoteImgRef, imgIndex)

		res := bldr.OCIStepResource().WithImgRef(remoteImgRef).Build()
		ociFetcher := oci.NewOCIFetcher(t.TempDir())

		stepCache := cache.NewWithOptions(cache.WithOCIFetcher(ociFetcher))
		specDef, err := stepCache.Get(context.Background(), t.TempDir(), res)
		require.NoError(t, err)
		require.Equal(t, []string{"bash"}, specDef.Definition.Exec.Command)
	})

	t.Run("loads OCI step in sub-directory", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		layer := bldr.
			OCIImageLayer(t).
			WithFile("/foo/bar/bob/step.yml", []byte("spec:\n---\nexec: {command: [bash]}")).
			Build()
		img := bldr.OCIImage(t).WithLayer(layer).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(remoteImgRef, imgIndex)

		res := bldr.
			OCIStepResource().
			WithImgRef(remoteImgRef).
			WithPath("foo", "bar", "bob").
			Build()
		ociFetcher := oci.NewOCIFetcher(t.TempDir())

		stepCache := cache.NewWithOptions(cache.WithOCIFetcher(ociFetcher))
		specDef, err := stepCache.Get(context.Background(), t.TempDir(), res)
		require.NoError(t, err)
		require.Equal(t, []string{"bash"}, specDef.Definition.Exec.Command)
	})

	t.Run("loads OCI step using a digest", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		layer := bldr.
			OCIImageLayer(t).
			WithFile("/step.yml", []byte("spec:\n---\nexec: {command: [sh]}")).
			Build()
		img := bldr.OCIImage(t).WithLayer(layer).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(registry.RefToImage("image", "latest"), imgIndex)

		digest, err := imgIndex.Digest()
		require.NoError(t, err)

		digestRef := registry.RefToImageDigest("image", digest)
		res := bldr.OCIStepResource().WithImgRef(digestRef).Build()
		ociFetcher := oci.NewOCIFetcher(t.TempDir())

		stepCache := cache.NewWithOptions(cache.WithOCIFetcher(ociFetcher))
		specDef, err := stepCache.Get(context.Background(), t.TempDir(), res)
		require.NoError(t, err)
		require.Equal(t, []string{"sh"}, specDef.Definition.Exec.Command)
	})

	t.Run("runs publish dist step", func(t *testing.T) {
		res := bldr.DistStepResource().WithStep("oci/publish").Build()
		distFetcher := dist.NewFetcher(stepdist.FindDistributedStep)

		stepCache := cache.NewWithOptions(cache.WithDistFetcher(distFetcher))
		specDef, err := stepCache.Get(context.Background(), t.TempDir(), res)
		require.NoError(t, err)
		require.Contains(t, strings.Join(specDef.Definition.Exec.Command, " "), "run")
	})
}
