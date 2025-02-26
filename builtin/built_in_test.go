package builtin_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/builtin"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestFindBuiltInStep(t *testing.T) {
	t.Run("returns files embedded in step", func(t *testing.T) {
		embeddedFS := bldr.Files(t).
			WriteFile("/bin/my_steps/step/files/hello.txt", "hello world").
			BuildFS()

		stepFS, err := builtin.FindBuiltInStep("my_steps/step", builtin.WithFileSystem(embeddedFS))
		require.NoError(t, err)

		helloTxt, err := stepFS.Open("files/hello.txt")
		require.NoError(t, err)
		defer helloTxt.Close()

		helloTxtContents, err := io.ReadAll(helloTxt)
		require.NoError(t, err)
		require.NoError(t, helloTxt.Close())
		require.Equal(t, "hello world", string(helloTxtContents))
	})

	t.Run("prevents path traversal", func(t *testing.T) {
		tests := []struct {
			name string
			step string
			err  string
		}{
			{
				name: "returns error if step not found",
				step: "my_steps/step",
				err:  `built-in step "my_steps/step" not found`,
			},
			{
				name: "cannot ask for step in current directory",
				step: ".",
				err:  `built-in step "." not found`,
			},
			{
				name: "cannot ask for step in current directory using ..",
				step: "my_step/step/../..",
				err:  `built-in step "my_step/step/../.." not found`,
			},
			{
				name: "cannot ask for step using ..",
				step: "..",
				err:  `built-in step ".." not found`,
			},
			{
				name: "cannot ask for step in previous sub directory",
				step: "../..",
				err:  `loading built-in step "../..": stat ..: invalid argument`,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				embeddedFS := bldr.Files(t).WriteFile("/bin/.keep", "").BuildFS()

				_, err := builtin.FindBuiltInStep(test.step, builtin.WithFileSystem(embeddedFS))
				require.Error(t, err)
				require.Contains(t, err.Error(), test.err)
			})
		}
	})
}
