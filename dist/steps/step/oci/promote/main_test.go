package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/require"

	fetchApi "gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/fetch/api"
	fetchBldr "gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/fetch/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/testutil/bldr"
	mainBldr "gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestRun(t *testing.T) {
	t.Run("promotes built step image", func(t *testing.T) {
		registry := fetchBldr.StartOCIRegistryServer(t)
		builtImageRef := registry.RefToImage("build/image", "1234")

		img := fetchBldr.OCIImage(t).WithFile("steps/my-step/step.yml", []byte("spec:\n---\nexec: {command: [bash]}")).Build()
		imgIndex := fetchBldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(builtImageRef, imgIndex)

		cliArgs, getEnv := bldr.CLIInputs().
			WithFromImage(builtImageRef.Name()).
			WithToRegistry(registry.Address()).
			WithToRepository("published/image").
			WithToVersion("1.0.1").
			Build()

		err := run(cliArgs, getEnv)
		require.NoError(t, err)

		publishedImgRef := registry.RefToImage("published/image", "1")
		imageDir := fetch(t, publishedImgRef, mainBldr.OCIPlatform.ThisPlatform)

		stepYaml, err := os.ReadFile(filepath.Join(imageDir, "steps", "my-step", "step.yml"))
		require.NoError(t, err)
		require.Equal(t, "spec:\n---\nexec: {command: [bash]}", string(stepYaml))
	})
}

func fetch(t *testing.T, imgRef name.Reference, forPlatforms ...*v1.Platform) string {
	imageDir, err := fetchApi.NewClient(t.TempDir()).Pull(t.Context(), imgRef, fetchApi.WithPlatforms(forPlatforms...))
	require.NoError(t, err)

	return imageDir
}
