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
	source, err := os.Executable()
	require.NoError(t, err)

	compareFiles := func(source, destination string) {
		dst, err := os.ReadFile(path.Join(destination, "step-runner"))
		require.NoError(t, err)

		src, err := os.ReadFile(source)
		require.NoError(t, err)

		assert.Equal(t, src, dst)
	}

	t.Run("copies file to destination directory idempotently", func(t *testing.T) {
		destination := t.TempDir()

		require.NoError(t, run(source, destination))
		compareFiles(source, destination)

		require.NoError(t, run(source, destination))
		compareFiles(source, destination)
	})

	t.Run("destination does not exist", func(t *testing.T) {
		destination := t.TempDir()

		require.NoError(t, run(source, destination))
		compareFiles(source, destination)
	})

	t.Run("destination is not a directory", func(t *testing.T) {
		_, destination, _, ok := runtime.Caller(0)
		require.True(t, ok)

		err := run("", destination)
		require.Error(t, run("", destination))
		require.Contains(t, err.Error(), fmt.Sprintf("mkdir %s: not a directory", destination))
	})

	t.Run("destination file exists", func(t *testing.T) {
		destination := t.TempDir()
		file, err := os.Create(path.Join(destination, "step-runner"))
		require.NoError(t, err)
		require.NoError(t, file.Close())

		err = run(source, destination)
		require.NoError(t, err)

		compareFiles(source, destination)
	})

	t.Run("destination file exists and is dir", func(t *testing.T) {
		destination := t.TempDir()
		err := os.Mkdir(path.Join(destination, "step-runner"), 0o755)
		require.NoError(t, err)

		err = run(source, destination)
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf(": open %s: is a directory", path.Join(destination, "step-runner")))
	})
}
