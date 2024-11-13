package bootstrap

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_run(t *testing.T) {
	t.Run("copies file to destination directory", func(t *testing.T) {
		source, err := os.Executable()
		require.NoError(t, err)

		destination := t.TempDir()

		require.NoError(t, run(source, destination))

		dst, err := os.ReadFile(path.Join(destination, "step-runner"))
		require.NoError(t, err)

		src, err := os.ReadFile(source)
		require.NoError(t, err)

		assert.Equal(t, src, dst)
	})

	t.Run("destination does not exists", func(t *testing.T) {
		err := run("", path.Join("foo", "bar", "baz"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "stat foo/bar/baz: no such file or directory")
	})

	t.Run("destination is not a directory", func(t *testing.T) {
		_, destination, _, ok := runtime.Caller(0)
		require.True(t, ok)

		err := run("", destination)
		require.Error(t, run("", destination))
		require.Contains(t, err.Error(), fmt.Sprintf("destination %q is not a directory", destination))
	})

	t.Run("destination file exists", func(t *testing.T) {
		tempDir := t.TempDir()
		file, err := os.Create(path.Join(tempDir, "step-runner"))
		require.NoError(t, err)
		require.NoError(t, file.Close())

		err = run("", tempDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("destination %q already exists", file.Name()))
	})
}
