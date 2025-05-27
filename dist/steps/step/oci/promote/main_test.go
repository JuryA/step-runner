package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	fetchBldr "gitlab.com/gitlab-org/step-runner/dist/steps/oci/fetch/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/testutil/bldr"
)

func TestRun(t *testing.T) {
	t.Run("promotes built step image", func(t *testing.T) {
		registry := fetchBldr.StartOCIRegistryServer(t)
		builtImageRef := registry.RefToImage("build/image", "1234")

		img := fetchBldr.OCIImage(t).WithFile("/step.yml", []byte("spec:\n---\nexec: {command: [bash, -c, echo FooBar]}")).Build()
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
	})
}
