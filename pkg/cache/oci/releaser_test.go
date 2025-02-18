package oci_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			WriteFile("dist/linux/amd64/program", []byte{1, 2, 3}).
			Build()

		downloadDir := t.TempDir()
		err := oci.NewReleaser(downloadDir).Release(ctx, remoteImgRef, baseDir)
		require.NoError(t, err)

		imageURL := strings.TrimSuffix(remoteImgRef.Name(), ":"+remoteImgRef.Identifier())
		fetcher := oci.NewOCIFetcher(downloadDir)
		imageDir, err := fetcher.Fetch(ctx, imageURL, remoteImgRef.Identifier(), internal.WithPlatforms(bldr.OCIPlatform.LinuxAMD64))
		require.NoError(t, err)

		stat, err := os.Stat(filepath.Join(imageDir, "step.yml"))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), stat.Mode())

		stepFile, err := os.ReadFile(filepath.Join(imageDir, "step.yml"))
		require.NoError(t, err)
		require.Equal(t, "spec:", string(stepFile))

		programFile, err := os.ReadFile(filepath.Join(imageDir, "program"))
		require.NoError(t, err)
		require.Equal(t, []byte{1, 2, 3}, programFile)
	})
}
