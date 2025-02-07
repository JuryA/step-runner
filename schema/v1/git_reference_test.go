package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitReference_Unmarshal(t *testing.T) {
	t.Run("unmarshals git reference", func(t *testing.T) {
		json := `{
			"dir": "/steps/step_a",
			"rev": "v2",
			"url": "gitlab.com/project",
			"file": "/path/to/file" }`

		var ref GitReference
		err := ref.UnmarshalJSON([]byte(json))
		require.NoError(t, err)
		require.Equal(t, "/steps/step_a", *ref.Dir)
		require.Equal(t, "v2", ref.Rev)
		require.Equal(t, "gitlab.com/project", ref.Url)
		require.Equal(t, "/path/to/file", *ref.File)
	})

	t.Run("fails to unmarshal when no revision", func(t *testing.T) {
		json := `{ "url": "gitlab.com/project" }`

		var ref GitReference
		err := ref.UnmarshalJSON([]byte(json))
		require.Error(t, err)
		require.Contains(t, err.Error(), "field rev in git: required")
	})

	t.Run("fails to unmarshal when no url", func(t *testing.T) {
		json := `{ "rev": "v2" }`

		var ref GitReference
		err := ref.UnmarshalJSON([]byte(json))
		require.Error(t, err)
		require.Contains(t, err.Error(), "field url in git: required")
	})
}
