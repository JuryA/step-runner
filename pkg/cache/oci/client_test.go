package oci_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestOCIRegistry(t *testing.T) {
	t.Run("downloads image to local directory", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		img := bldr.OCIImage(t).WithFile("/my-steps/step.yml", []byte("Hello, world")).Build()
		registry.Push(remoteImgRef, img)

		imageDir, err := oci.NewClient(t.TempDir()).Pull(context.Background(), remoteImgRef)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(imageDir, "my-steps", "step.yml"))
		require.NoError(t, err)
		require.Equal(t, "Hello, world", string(content))
	})

	t.Run("fails if image does not exist on registry", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		_, err := oci.NewClient(t.TempDir()).Pull(context.Background(), remoteImgRef)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot find remote OCI image matching local platform")
	})
}
