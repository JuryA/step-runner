package api_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"

	fetchApi "gitlab.com/gitlab-org/step-runner/dist/steps/oci/fetch/api"

	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/build/api"

	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/build/internal"
	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/build/testutil/bldr"

	mainBldr "gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestReleaser_Release(t *testing.T) {
	t.Run("builds an image specific for an architecture", func(t *testing.T) {
		registry := mainBldr.StartOCIRegistryServer(t)
		remoteImgRef := bldr.RemoteImageRef(t).WithRegistry(registry.Address()).Build()

		baseDir := mainBldr.Files(t).
			WriteFileWithPerms("dist/common/step.yml", "spec:", 0600).
			WriteFile("dist/linux/amd64/program", "123").
			Build()

		common := bldr.OCIArtifact(t).
			Generic().
			WithFrom(filepath.Join(baseDir, "dist/common")).
			WithTo("/my_step").
			BuildArtifacts()

		platformSpecific := bldr.OCIArtifact(t).
			LinuxAMD64().
			WithFrom(filepath.Join(baseDir, "dist/linux/amd64/program")).
			WithTo("/my_step/program").
			BuildArtifacts()

		imageIndex, err := api.NewReleaser().Release(t.Context(), remoteImgRef, common, platformSpecific)
		require.NoError(t, err)
		require.NotNil(t, imageIndex)

		imageDir := fetch(t, remoteImgRef, mainBldr.OCIPlatform.LinuxAMD64)
		require.Equal(t, "spec:", readFile(t, filepath.Join(imageDir, "my_step", "step.yml")))
		require.Equal(t, "123", readFile(t, filepath.Join(imageDir, "my_step", "program")))
	})

	t.Run("builds an image with optional platform settings", func(t *testing.T) {
		registry := mainBldr.StartOCIRegistryServer(t)
		remoteImgRef := bldr.RemoteImageRef(t).WithRegistry(registry.Address()).Build()

		platform := &v1.Platform{
			Architecture: "amd64",
			OS:           "windows",
			OSVersion:    "10.0.26100.3476",
			OSFeatures:   []string{"win32k"},
			Variant:      "v7",
			Features:     []string{"gpu"},
		}
		platformSpecific := bldr.OCIArtifact(t).WithPlatform(platform).BuildArtifacts()

		_, err := api.NewReleaser().Release(t.Context(), remoteImgRef, internal.NewArtifacts(), platformSpecific)
		require.NoError(t, err)

		imgIndex, err := remote.Index(remoteImgRef)
		require.NoError(t, err)

		indexManifest, err := imgIndex.IndexManifest()
		require.NoError(t, err)
		require.Len(t, indexManifest.Manifests, 1)
		require.Equal(t, "windows", indexManifest.Manifests[0].Platform.OS)
		require.Equal(t, "10.0.26100.3476", indexManifest.Manifests[0].Platform.OSVersion)
		require.Equal(t, []string{"win32k"}, indexManifest.Manifests[0].Platform.OSFeatures)
		require.Equal(t, "amd64", indexManifest.Manifests[0].Platform.Architecture)
		require.Equal(t, "v7", indexManifest.Manifests[0].Platform.Variant)
		require.Equal(t, []string{"gpu"}, indexManifest.Manifests[0].Platform.Features)
	})

	t.Run("normalizes os/architecture names", func(t *testing.T) {
		registry := mainBldr.StartOCIRegistryServer(t)
		remoteImgRef := bldr.RemoteImageRef(t).WithRegistry(registry.Address()).Build()

		platform := &v1.Platform{Architecture: "AARCH64", OS: "linux"}
		platformSpecific := bldr.OCIArtifact(t).WithPlatform(platform).BuildArtifacts()

		_, err := api.NewReleaser().Release(t.Context(), remoteImgRef, internal.NewArtifacts(), platformSpecific)
		require.NoError(t, err)

		imgIndex, err := remote.Index(remoteImgRef)
		require.NoError(t, err)

		indexManifest, err := imgIndex.IndexManifest()
		require.NoError(t, err)
		require.Len(t, indexManifest.Manifests, 1)
		require.Equal(t, "linux", indexManifest.Manifests[0].Platform.OS)
		require.Equal(t, "arm64", indexManifest.Manifests[0].Platform.Architecture)
	})

	t.Run("builds images for many architectures", func(t *testing.T) {
		registry := mainBldr.StartOCIRegistryServer(t)
		remoteImgRef := bldr.RemoteImageRef(t).WithRegistry(registry.Address()).Build()

		baseDir := mainBldr.Files(t).
			WriteFile("dist/common/step.yml", "spec:").
			WriteFile("dist/linux/amd64/program", "amd64").
			WriteFile("dist/linux/arm64/program", "arm64").
			Build()

		common := bldr.OCIArtifact(t).
			Generic().
			WithFrom(filepath.Join(baseDir, "dist/common")).
			WithTo("/").
			BuildArtifacts()

		platformSpecific := internal.NewArtifacts(
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

		_, err := api.NewReleaser().Release(t.Context(), remoteImgRef, common, platformSpecific)
		require.NoError(t, err)

		amd64Dir := fetch(t, remoteImgRef, mainBldr.OCIPlatform.LinuxAMD64)
		require.Equal(t, "spec:", readFile(t, filepath.Join(amd64Dir, "step.yml")))
		require.Equal(t, "amd64", readFile(t, filepath.Join(amd64Dir, "program")))

		arm64Dir := fetch(t, remoteImgRef, mainBldr.OCIPlatform.LinuxARM64)
		require.Equal(t, "spec:", readFile(t, filepath.Join(arm64Dir, "step.yml")))
		require.Equal(t, "arm64", readFile(t, filepath.Join(arm64Dir, "program")))
	})

	t.Run("writes an empty directory", func(t *testing.T) {
		registry := mainBldr.StartOCIRegistryServer(t)
		remoteImgRef := bldr.RemoteImageRef(t).WithRegistry(registry.Address()).Build()

		baseDir := mainBldr.Files(t).WriteDir("/my/files").Build()

		platformSpecific := bldr.OCIArtifact(t).
			LinuxAMD64().
			WithFrom(filepath.Join(baseDir, "my", "files")).
			WithTo("/app/my/files").
			BuildArtifacts()

		_, err := api.NewReleaser().Release(t.Context(), remoteImgRef, internal.NewArtifacts(), platformSpecific)
		require.NoError(t, err)

		imageDir := fetch(t, remoteImgRef, mainBldr.OCIPlatform.LinuxAMD64)
		myFiles := filepath.Join(imageDir, "app", "my", "files")

		stat, err := os.Stat(myFiles)
		require.NoError(t, err)
		require.True(t, stat.Mode().IsDir())

		entries, err := os.ReadDir(myFiles)
		require.NoError(t, err)
		require.Len(t, entries, 0)
	})

	t.Run("fails to build when image with version already exists", func(t *testing.T) {
		registry := mainBldr.StartOCIRegistryServer(t)
		remoteImgRef := bldr.RemoteImageRef(t).WithRegistry(registry.Address()).Build()

		registry.Push(remoteImgRef, mainBldr.OCIImage(t).Build())

		platformSpecific := bldr.OCIArtifact(t).BuildArtifacts()
		_, err := api.NewReleaser().Release(t.Context(), remoteImgRef, internal.NewArtifacts(), platformSpecific)
		require.Error(t, err)
		require.Equal(t, fmt.Sprintf("image already published: %s", remoteImgRef), err.Error())
	})
}

func readFile(t *testing.T, path string) string {
	require.FileExists(t, path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(data)
}

func fetch(t *testing.T, imgRef name.Reference, forPlatforms ...*v1.Platform) string {
	imageDir, err := fetchApi.NewClient(t.TempDir()).Pull(t.Context(), imgRef, fetchApi.WithPlatforms(forPlatforms...))
	require.NoError(t, err)

	return imageDir
}
