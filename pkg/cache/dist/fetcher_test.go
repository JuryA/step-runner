package dist_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	stepdist "gitlab.com/gitlab-org/step-runner/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestFetcher_Fetch(t *testing.T) {
	t.Run("writes step files to disk", func(t *testing.T) {
		embeddedFS := bldr.Files(t).
			WriteFile("files/hello.txt", "hello world").
			BuildFS()

		fetcher := dist.NewFetcher(alwaysReturnsFS(embeddedFS))
		t.Cleanup(fetcher.CleanUp)

		baseDir, err := fetcher.Fetch("my_steps/step")
		require.NoError(t, err)

		helloWorldPath := filepath.Join(baseDir, "my_steps", "step", "files", "hello.txt")

		helloWorldData, err := os.ReadFile(helloWorldPath)
		require.NoError(t, err)
		require.Equal(t, "hello world", string(helloWorldData))
	})

	t.Run("caches steps written to disk", func(t *testing.T) {
		embeddedFS := bldr.Files(t).WriteFile("step.yml", "spec:").BuildFS()
		fetcher := dist.NewFetcher(alwaysReturnsFS(embeddedFS))
		t.Cleanup(fetcher.CleanUp)

		baseDirA, err := fetcher.Fetch("my_step")
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(baseDirA, "my_step", "step.yml"))

		baseDirB, err := fetcher.Fetch("my_step")
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(baseDirB, "my_step", "step.yml"))
		require.Equal(t, baseDirA, baseDirB)
	})

	t.Run("errors on step not found", func(t *testing.T) {
		fetcher := dist.NewFetcher(stepdist.FindDistributedStep)
		t.Cleanup(fetcher.CleanUp)

		_, err := fetcher.Fetch("invalid/step")
		require.Error(t, err)
		require.Contains(t, err.Error(), `fetch: distributed step "invalid/step" not found`)
	})

	t.Run("files in the root named run are executable", func(t *testing.T) {
		tests := []struct {
			name     string
			filename string
			expected string
		}{
			{
				name:     "compiled go programs are executable",
				filename: "run",
				expected: "-r-xr-xr-x",
			},
			{
				name:     "file ending with run is not executable",
				filename: "my_run",
				expected: "-r--r--r--",
			}, {
				name:     "windows executable are executable",
				filename: "run.exe",
				expected: "-r-xr-xr-x",
			},
			{
				name:     "scripts are executable",
				filename: "my_script.sh",
				expected: "-r-xr-xr-x",
			},
			{
				name:     "all other files are read only",
				filename: "templates/index.html.template",
				expected: "-r--r--r--",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				embeddedFS := bldr.Files(t).WriteFileWithPerms(test.filename, "exec me", 0444).BuildFS()
				fetcher := dist.NewFetcher(alwaysReturnsFS(embeddedFS))
				t.Cleanup(fetcher.CleanUp)

				baseDir, err := fetcher.Fetch("my_step")
				require.NoError(t, err)

				info, err := os.Stat(filepath.Join(baseDir, "my_step", test.filename))
				require.NoError(t, err)
				require.Equal(t, test.expected, info.Mode().String())
			})
		}
	})
}

func alwaysReturnsFS(value fs.FS) stepdist.StepFinder {
	return func(step string, options ...func(*stepdist.FindStepsOptions)) (fs.FS, error) {
		return value, nil
	}
}
