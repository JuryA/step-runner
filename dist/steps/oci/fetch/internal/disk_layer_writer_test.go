package internal_test

import (
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/dist-steps/oci/fetch/internal/testutil/bldr"

	"gitlab.com/gitlab-org/dist-steps/oci/fetch/internal"
)

func TestDiskLayerWriter(t *testing.T) {
	t.Run("writes files and directories to disk", func(t *testing.T) {
		layer := bldr.OCIImageLayer(t).WithFile("/path/to/file", []byte("foobar")).Build()
		dir := t.TempDir()

		err := internal.NewDiskLayerWriter().Write([]v1.Layer{layer}, dir)
		require.NoError(t, err)

		fileContent, err := os.ReadFile(filepath.Join(dir, "path/to/file"))
		require.NoError(t, err)
		require.Equal(t, []byte("foobar"), fileContent)
	})

	t.Run("files are read only", func(t *testing.T) {
		layer := bldr.OCIImageLayer(t).WithFileWithPerms("/my-file", []byte{}, 0755).Build()
		dir := t.TempDir()

		err := internal.NewDiskLayerWriter().Write([]v1.Layer{layer}, dir)
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(dir, "/my-file"))
		require.NoError(t, err)
		require.Equal(t, "-r--------", info.Mode().String())
	})

	t.Run("writes empty files to disk", func(t *testing.T) {
		layer := bldr.OCIImageLayer(t).WithFile("/my-file", []byte{}).Build()
		dir := t.TempDir()

		err := internal.NewDiskLayerWriter().Write([]v1.Layer{layer}, dir)
		require.NoError(t, err)

		fileContent, err := os.ReadFile(filepath.Join(dir, "/my-file"))
		require.NoError(t, err)
		require.Equal(t, []byte{}, fileContent)
	})
}
