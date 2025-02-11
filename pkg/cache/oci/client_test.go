package oci_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/platforms"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestOCIRegistry_Pull_Image(t *testing.T) {
	t.Run("downloads image and step to local directory", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		img := bldr.OCIImage(t).WithFile("/my-steps/step.yml", []byte("Hello, world")).Build()
		imgIndex := bldr.OCIImageIndex(t).WithPlatformImage(bldr.OCIPlatform.Generic, img).Build()
		registry.PushImageIndex(remoteImgRef, imgIndex)

		client := oci.NewClient(t.TempDir())
		imageDir, err := client.Pull(context.Background(), remoteImgRef, WithPlatforms(bldr.OCIPlatform.Generic))
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(imageDir, "my-steps", "step.yml"))
		require.NoError(t, err)
		require.Equal(t, "Hello, world", string(content))
	})

	t.Run("fails if image is not an an image index", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		registry.Push(remoteImgRef, bldr.OCIImage(t).WithEmptyFile("/my-file").Build())

		client := oci.NewClient(t.TempDir())
		_, err := client.Pull(context.Background(), remoteImgRef, WithPlatforms(bldr.OCIPlatform.Generic))
		require.Error(t, err)
		require.Contains(t, err.Error(), "fetching index: unexpected media type for ImageIndex(): application/vnd.docker.distribution.manifest.v2+json; call Image() instead")
	})

	t.Run("fails if image does not exist on registry", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		_, err := oci.NewClient(t.TempDir()).Pull(context.Background(), remoteImgRef)
		require.Error(t, err)
		require.Contains(t, err.Error(), "MANIFEST_UNKNOWN: manifest unknown; unknown tag=latest")
	})
}

func TestOCIRegistry_Pull_Platforms(t *testing.T) {
	tests := []struct {
		name             string
		imgIndex         v1.ImageIndex
		downloadFor      []*v1.Platform
		expectFileExists string
		expectError      string
	}{
		{
			name: "downloads linux amd64",
			imgIndex: bldr.OCIImageIndex(t).
				WithPlatformImage(bldr.OCIPlatform.LinuxAMD64, bldr.OCIImage(t).WithEmptyFile("/amd64").Build()).
				Build(),
			downloadFor:      []*v1.Platform{bldr.OCIPlatform.LinuxAMD64},
			expectFileExists: "/amd64",
		},
		{
			name: "downloads windows amd64",
			imgIndex: bldr.OCIImageIndex(t).
				WithPlatformImage(bldr.OCIPlatform.WindowsAMD64, bldr.OCIImage(t).WithEmptyFile("/win").Build()).
				Build(),
			downloadFor:      []*v1.Platform{bldr.OCIPlatform.WindowsAMD64},
			expectFileExists: "/win",
		},
		{
			name: "downloads linux arm64",
			imgIndex: bldr.OCIImageIndex(t).
				WithPlatformImage(bldr.OCIPlatform.LinuxARM64, bldr.OCIImage(t).WithEmptyFile("/arm64").Build()).
				Build(),
			downloadFor:      []*v1.Platform{bldr.OCIPlatform.LinuxARM64},
			expectFileExists: "/arm64",
		},
		{
			name: "downloads linux arm64 if v8 isn't available",
			imgIndex: bldr.OCIImageIndex(t).
				WithPlatformImage(bldr.OCIPlatform.LinuxARM64, bldr.OCIImage(t).WithEmptyFile("/arm64").Build()).
				Build(),
			downloadFor:      []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8},
			expectFileExists: "/arm64",
		},
		{
			name: "downloads linux arm64v8 if arm64 isn't available",
			imgIndex: bldr.OCIImageIndex(t).
				WithPlatformImage(bldr.OCIPlatform.LinuxARM64v8, bldr.OCIImage(t).WithEmptyFile("/arm64v8").Build()).
				Build(),
			downloadFor:      []*v1.Platform{bldr.OCIPlatform.LinuxARM64},
			expectFileExists: "/arm64v8",
		},
		{
			name: "falls back to generic",
			imgIndex: bldr.OCIImageIndex(t).
				WithPlatformImage(bldr.OCIPlatform.Generic, bldr.OCIImage(t).WithEmptyFile("/generic").Build()).
				Build(),
			downloadFor:      []*v1.Platform{bldr.OCIPlatform.LinuxARM64v7, bldr.OCIPlatform.Generic},
			expectFileExists: "/generic",
		},
		{
			name:        "prints all platforms in error message",
			imgIndex:    bldr.OCIImageIndex(t).WithPlatformImage(bldr.OCIPlatform.WindowsAMD64, bldr.OCIImage(t).Build()).Build(),
			downloadFor: []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8, bldr.OCIPlatform.Generic},
			expectError: "didn't find an image matching platform linux/arm64/v8 or generic",
		},
	}

	ctx := context.Background()
	registry := bldr.StartOCIRegistryServer(t)
	remoteImgRef := registry.RefToImage("my-image", "latest")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry.PushImageIndex(remoteImgRef, test.imgIndex)

			client := oci.NewClient(t.TempDir())
			imageDir, err := client.Pull(ctx, remoteImgRef, WithPlatforms(test.downloadFor...))

			if test.expectError == "" {
				require.NoError(t, err)
				require.FileExists(t, filepath.Join(imageDir, test.expectFileExists))
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectError)
			}
		})
	}
}

func WithPlatforms(v1Platforms ...*v1.Platform) func(*oci.PullOption) {
	return func(opt *oci.PullOption) {
		opt.Platforms = make([]platforms.Platform, len(v1Platforms))

		for i := range v1Platforms {
			opt.Platforms[i] = oci.ConvertPlatformV1ToCtrd(v1Platforms[i])
		}
	}
}
