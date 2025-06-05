package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/fetch/internal"
)

func TestOutputs_Write(t *testing.T) {
	t.Run("writes outputs to file", func(t *testing.T) {
		imgRef := name.MustParseReference("registry.gitlab.com:8080/my-group/my-project/image:10.1.1")
		outputFile := filepath.Join(t.TempDir(), "output.txt")

		err := internal.NewOutputs(outputFile).Write("/path/to/step_dir", imgRef, "step.yml")
		require.NoError(t, err)

		fileData, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		lines := strings.Split(string(fileData), "\n")
		require.Equal(t, `{"name":"fetched_step_path","value":"/path/to/step_dir/step.yml"}`, lines[0])
		require.Equal(t, `{"name":"ref","value":"registry.gitlab.com:8080/my-group/my-project/image:10.1.1"}`, lines[1])
	})
}
