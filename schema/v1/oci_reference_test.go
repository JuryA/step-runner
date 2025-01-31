package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOCIReference_Unmarshal(t *testing.T) {
	t.Run("unmarshals oci reference", func(t *testing.T) {
		json := `{
			"url": "registry.gitlab.com/project",
			"tag": "3.0.0" }`

		var ref OCIReference
		err := ref.UnmarshalJSON([]byte(json))
		require.NoError(t, err)
		require.Equal(t, "registry.gitlab.com/project", ref.Url)
		require.Equal(t, "3.0.0", ref.Tag)
	})

	t.Run("fails to unmarshal when no tag", func(t *testing.T) {
		json := `{ "url": "registry.gitlab.com/project" }`

		var ref OCIReference
		err := ref.UnmarshalJSON([]byte(json))
		require.Error(t, err)
		require.Contains(t, err.Error(), "field tag in oci: required")
	})

	t.Run("fails to unmarshal when no url", func(t *testing.T) {
		json := `{ "tag": "3.0.0" }`

		var ref OCIReference
		err := ref.UnmarshalJSON([]byte(json))
		require.Error(t, err)
		require.Contains(t, err.Error(), "field url in oci: required")
	})
}
