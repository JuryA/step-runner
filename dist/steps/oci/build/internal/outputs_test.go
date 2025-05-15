package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/dist/steps/oci/build/internal"

	mainBldr "gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestOutputs_Write(t *testing.T) {
	t.Run("writes ref to file", func(t *testing.T) {
		imgRef := name.MustParseReference("registry.gitlab.com:8080/my-group/my-project/image:10.1.1")
		outputFile := filepath.Join(t.TempDir(), "output.txt")

		err := internal.NewOutputs(outputFile).Write(imgRef, mainBldr.OCIImageIndex(t).Build())
		require.NoError(t, err)

		fileData, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		lines := strings.Split(string(fileData), "\n")
		require.Equal(t, `{"name":"registry","value":"registry.gitlab.com:8080"}`, lines[0])
		require.Equal(t, `{"name":"repository","value":"my-group/my-project/image"}`, lines[1])
		require.Equal(t, `{"name":"tag","value":"10.1.1"}`, lines[2])
		require.Equal(t, `{"name":"ref","value":"registry.gitlab.com:8080/my-group/my-project/image:10.1.1"}`, lines[3])
		require.Regexp(t, `{"name":"digest","value":{"algorithm":"sha256","hash":"[a-f0-9]{64}","value":"sha256:[a-f0-9]{64}"}}`, lines[4])
	})
}
