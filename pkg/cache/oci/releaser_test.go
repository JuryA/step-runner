package oci_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestReleaser_Release(t *testing.T) {
	t.Run("publishes an image specific for an architecture", func(t *testing.T) {
		ctx := context.Background()
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		baseDir := bldr.Files(t).
			WriteFileWithPerms("dist/common/step.yml", "spec:", 0600).
			WriteFile("dist/linux/amd64/program", "123").
			Build()

		artifacts := oci.NewArtifacts(
			bldr.OCIArtifact(t).
				Generic().
				WithFrom(filepath.Join(baseDir, "dist/common")).
				WithTo("/my_step").
				Build(),
			bldr.OCIArtifact(t).
				LinuxAMD64().
				WithFrom(filepath.Join(baseDir, "dist/linux/amd64/program")).
				WithTo("/my_step/program").
				Build())

		downloadDir := t.TempDir()
		err := oci.NewReleaser(downloadDir).Release(ctx, remoteImgRef, artifacts)
		require.NoError(t, err)

		imageDir := fetch(t, remoteImgRef, bldr.OCIPlatform.LinuxAMD64)
		stat, err := os.Stat(filepath.Join(imageDir, "my_step", "step.yml"))
		require.NoError(t, err)
		require.Equal(t, "-rw-r--r--", stat.Mode().String())

		require.Equal(t, "spec:", readFile(t, filepath.Join(imageDir, "my_step", "step.yml")))
		require.Equal(t, "123", readFile(t, filepath.Join(imageDir, "my_step", "program")))
	})

	t.Run("publishes images for many architectures", func(t *testing.T) {
		ctx := context.Background()
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		baseDir := bldr.Files(t).
			WriteFile("dist/common/step.yml", "spec:").
			WriteFile("dist/linux/amd64/program", "amd64").
			WriteFile("dist/linux/arm64/program", "arm64").
			Build()

		artifacts := oci.NewArtifacts(
			bldr.OCIArtifact(t).
				Generic().
				WithFrom(filepath.Join(baseDir, "dist/common")).
				WithTo("/").
				Build(),
			bldr.OCIArtifact(t).
				LinuxAMD64().
				WithFrom(filepath.Join(baseDir, "dist/linux/amd64")).
				WithTo("/").
				Build(),
			bldr.OCIArtifact(t).
				LinuxARM64().
				WithFrom(filepath.Join(baseDir, "dist/linux/arm64")).
				WithTo("/").
				Build())

		downloadDir := t.TempDir()
		err := oci.NewReleaser(downloadDir).Release(ctx, remoteImgRef, artifacts)
		require.NoError(t, err)

		amd64Dir := fetch(t, remoteImgRef, bldr.OCIPlatform.LinuxAMD64)
		require.Equal(t, "spec:", readFile(t, filepath.Join(amd64Dir, "step.yml")))
		require.Equal(t, "amd64", readFile(t, filepath.Join(amd64Dir, "program")))

		arm64Dir := fetch(t, remoteImgRef, bldr.OCIPlatform.LinuxARM64)
		require.Equal(t, "spec:", readFile(t, filepath.Join(arm64Dir, "step.yml")))
		require.Equal(t, "arm64", readFile(t, filepath.Join(arm64Dir, "program")))
	})
}

func readFile(t *testing.T, path string) string {
	require.FileExists(t, path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(data)
}

func fetch(t *testing.T, imgRef name.Reference, forPlatforms ...*v1.Platform) string {
	platform := internal.WithPlatforms(forPlatforms...)
	imageDir, err := oci.NewOCIFetcher(t.TempDir()).Fetch(context.Background(), imgRef, platform)
	require.NoError(t, err)

	return imageDir
}
