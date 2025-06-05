package remote_test

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	remoteRepo "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"

	fetchBldr "gitlab.com/gitlab-org/step-runner/dist/steps/oci/fetch/testutil/bldr"
	mainBldr "gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"

	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/internal/remote"
)

func TestCopy(t *testing.T) {
	t.Run("copies image from source to target repository", func(t *testing.T) {
		sourceRegistry := mainBldr.StartOCIRegistryServer(t)
		targetRegistry := mainBldr.StartOCIRegistryServer(t)

		tests := []struct {
			name       string
			fromImgRef name.Reference
			toImgRef   name.Reference
		}{
			{
				name:       "target registry differs from source",
				fromImgRef: sourceRegistry.RefToImage("tmp-img", "pipeline-1"),
				toImgRef:   targetRegistry.RefToImage("application", "1.0.2"),
			},
			{
				name:       "target registry is same as source",
				fromImgRef: sourceRegistry.RefToImage("tmp-img", "pipeline-2"),
				toImgRef:   sourceRegistry.RefToImage("app", "3.0.0"),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				fromImg := fetchBldr.OCIImage(t).WithFile("steps/my-step/step.yml", []byte("spec:")).Build()
				sourceRegistry.Push(test.fromImgRef, fromImg)

				err := remote.Copy(t.Context(), test.fromImgRef, test.toImgRef)
				require.NoError(t, err)

				toImg, err := remoteRepo.Image(test.toImgRef)
				require.NoError(t, err)

				fromDigest, err := fromImg.Digest()
				require.NoError(t, err)

				toDigest, err := toImg.Digest()
				require.NoError(t, err)
				require.Equal(t, fromDigest, toDigest)
			})
		}
	})

	t.Run("copies image index from source to target repository", func(t *testing.T) {
		sourceRegistry := mainBldr.StartOCIRegistryServer(t)
		targetRegistry := mainBldr.StartOCIRegistryServer(t)

		tests := []struct {
			name       string
			fromImgRef name.Reference
			toImgRef   name.Reference
		}{
			{
				name:       "target registry differs from source",
				fromImgRef: sourceRegistry.RefToImage("tmp-img", "pipeline-1"),
				toImgRef:   targetRegistry.RefToImage("application", "1.0.2"),
			},
			{
				name:       "target registry is same as source",
				fromImgRef: sourceRegistry.RefToImage("tmp-img", "pipeline-2"),
				toImgRef:   sourceRegistry.RefToImage("app", "3.0.0"),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				platImgA := fetchBldr.OCIImage(t).WithFile("step.yml", []byte("spec: 1")).Build()
				platImgB := fetchBldr.OCIImage(t).WithFile("step.yml", []byte("spec: 2")).Build()
				fromImg := fetchBldr.OCIImageIndex(t).
					WithPlatformImage(mainBldr.OCIPlatform.LinuxAMD64, platImgA).
					WithPlatformImage(mainBldr.OCIPlatform.LinuxARM64, platImgB).
					Build()
				sourceRegistry.PushImageIndex(test.fromImgRef, fromImg)

				err := remote.Copy(t.Context(), test.fromImgRef, test.toImgRef)
				require.NoError(t, err)

				toImg, err := remoteRepo.Index(test.toImgRef)
				require.NoError(t, err)

				fromDigest, err := fromImg.Digest()
				require.NoError(t, err)

				toDigest, err := toImg.Digest()
				require.NoError(t, err)
				require.Equal(t, fromDigest, toDigest)
			})
		}
	})
}
