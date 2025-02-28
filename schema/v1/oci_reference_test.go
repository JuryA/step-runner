package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOCIReference_Unmarshal(t *testing.T) {
	t.Run("unmarshals oci reference", func(t *testing.T) {
		json := `{
			"registry": "registry.gitlab.com",
			"repository": "group/project/image",
			"tag": "3.0.0",
			"dir": "/path/to/step",
			"file": "step.yml" }`

		var ref OCIReference
		err := ref.UnmarshalJSON([]byte(json))
		require.NoError(t, err)
		require.Equal(t, "registry.gitlab.com", ref.Registry)
		require.Equal(t, "group/project/image", ref.Repository)
		require.Equal(t, "3.0.0", ref.Tag)
		require.Equal(t, "/path/to/step", *ref.Dir)
		require.Equal(t, "step.yml", *ref.File)
	})

	t.Run("fails to unmarshal when no registry", func(t *testing.T) {
		json := `{
			"repository": "group/project/image",
			"tag": "3.0.0" }`

		var ref OCIReference
		err := ref.UnmarshalJSON([]byte(json))
		require.Error(t, err)
		require.Contains(t, err.Error(), "field registry in oci: required")
	})

	t.Run("fails to unmarshal when no repository", func(t *testing.T) {
		json := `{
			"registry": "registry.gitlab.com/project",
			"tag": "3.0.0" }`

		var ref OCIReference
		err := ref.UnmarshalJSON([]byte(json))
		require.Error(t, err)
		require.Contains(t, err.Error(), "field repository in oci: required")
	})

	t.Run("fails to unmarshal when no tag", func(t *testing.T) {
		json := `{
			"registry": "registry.gitlab.com/project",
			"repository": "group/project/image" }`

		var ref OCIReference
		err := ref.UnmarshalJSON([]byte(json))
		require.Error(t, err)
		require.Contains(t, err.Error(), "field tag in oci: required")
	})
}
