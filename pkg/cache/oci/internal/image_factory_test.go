package internal_test

import (
	"archive/tar"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestImageFactory_BuildLayer(t *testing.T) {
	t.Run("builds empty layer", func(t *testing.T) {
		archiveFS := bldr.Files(t).BuildFS()

		layer, err := internal.NewImageFactory().BuildLayer(archiveFS)
		require.NoError(t, err)

		uncompressed, err := layer.Uncompressed()
		require.NoError(t, err)

		untar := tar.NewReader(uncompressed)
		_, err = untar.Next()
		require.Equal(t, io.EOF, err)
	})

	t.Run("has steps media type", func(t *testing.T) {
		layer, err := internal.NewImageFactory().BuildLayer(bldr.Files(t).BuildFS())
		require.NoError(t, err)

		mediaType, err := layer.MediaType()
		require.NoError(t, err)
		require.Equal(t, "application/vnd.gitlab.step.layer.v1.tar+zstd", string(mediaType))
	})

	t.Run("archives using tar format", func(t *testing.T) {
		archiveFS := bldr.Files(t).
			WriteFile("/animals/sheep.txt", "how now brown cow").
			BuildFS()

		layer, err := internal.NewImageFactory().BuildLayer(archiveFS)
		require.NoError(t, err)

		mediaType, err := layer.MediaType()
		require.NoError(t, err)
		require.Equal(t, "application/vnd.gitlab.step.layer.v1.tar+zstd", string(mediaType))

		uncompressed, err := layer.Uncompressed()
		require.NoError(t, err)

		untar := tar.NewReader(uncompressed)
		next, err := untar.Next()
		require.NoError(t, err)
		require.Equal(t, "animals/sheep.txt", next.Name)

		data, err := io.ReadAll(untar)
		require.NoError(t, err)
		require.Equal(t, "how now brown cow", string(data))
	})

	t.Run("compresses using zstd", func(t *testing.T) {
		archiveFS := bldr.Files(t).WriteFile("/sheep.txt", "baa").BuildFS()

		layer, err := internal.NewImageFactory().BuildLayer(archiveFS)
		require.NoError(t, err)

		compressed, err := layer.Compressed()
		require.NoError(t, err)

		data, err := io.ReadAll(compressed)
		require.NoError(t, err)
		require.Equal(t, []byte{0x28, 0xB5, 0x2F, 0xFD}, data[:4]) // zstd start byte sequence
	})

	t.Run("can rebuild the same folder", func(t *testing.T) {
		archiveFS := bldr.Files(t).WriteFile("/sheep.txt", "baa").BuildFS()

		factory := internal.NewImageFactory()
		layerA, err := factory.BuildLayer(archiveFS)
		require.NoError(t, err)
		require.NotNil(t, layerA)

		layerB, err := factory.BuildLayer(archiveFS)
		require.NoError(t, err)
		require.NotNil(t, layerB)

		digestA, err := layerA.Digest()
		require.NoError(t, err)

		digestB, err := layerB.Digest()
		require.NoError(t, err)
		require.Equal(t, digestA, digestB)
	})
}

func TestImageFactory_BuildImage(t *testing.T) {
	createdAt, err := time.Parse(time.RFC3339, "2024-02-18T15:04:05Z")
	require.NoError(t, err)

	layerA := bldr.OCIImageLayer(t).WithFile("/foo", []byte("foo")).Build()
	layerB := bldr.OCIImageLayer(t).WithFile("/bar", []byte("bar")).Build()

	image, err := internal.NewImageFactory().BuildImage(createdAt, layerA, layerB)
	require.NoError(t, err)

	layers, err := image.Layers()
	require.NoError(t, err)
	require.Len(t, layers, 2)

	manifest, err := image.Manifest()
	require.NoError(t, err)
	require.Equal(t, "application/vnd.oci.image.manifest.v1+json", string(manifest.MediaType))
	require.Equal(t, "application/vnd.gitlab.step.image.v1", string(manifest.Config.MediaType))
	require.Equal(t, "2024-02-18T15:04:05Z", manifest.Annotations["org.opencontainers.image.created"])
}

func TestImageFactory_BuildImageIndex(t *testing.T) {
	createdAt, err := time.Parse(time.RFC3339, "2024-02-18T15:04:05Z")
	require.NoError(t, err)

	image := bldr.OCIImage(t).WithFile("/foo", []byte("foo")).Build()
	imagePlatform := internal.PlatformImage{Image: image, Platform: bldr.OCIPlatform.LinuxARM64v7}
	imageIndex := internal.NewImageFactory().BuildImageIndex(createdAt, imagePlatform)

	mediaType, err := imageIndex.MediaType()
	require.NoError(t, err)
	require.Equal(t, "application/vnd.oci.image.index.v1+json", string(mediaType))

	manifest, err := imageIndex.IndexManifest()
	require.NoError(t, err)
	require.Len(t, manifest.Manifests, 1)

	firstManifest := manifest.Manifests[0]
	require.Equal(t, "application/vnd.oci.image.manifest.v1+json", string(firstManifest.MediaType))
	require.Equal(t, "linux", firstManifest.Platform.OS)
	require.Equal(t, "arm64", firstManifest.Platform.Architecture)
	require.Equal(t, "v7", firstManifest.Platform.Variant)
	require.Equal(t, "2024-02-18T15:04:05Z", manifest.Annotations["org.opencontainers.image.created"])
}
