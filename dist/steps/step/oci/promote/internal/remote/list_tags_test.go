package remote

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	fetchBldr "gitlab.com/gitlab-org/step-runner/dist/steps/oci/fetch/testutil/bldr"
)

func TestListTags(t *testing.T) {
	t.Run("all the tags", func(t *testing.T) {
		registry := fetchBldr.StartOCIRegistryServer(t)
		img := fetchBldr.OCIImage(t).Build()

		registry.Push(registry.RefToImage("build/image", "3"), img)
		registry.Push(registry.RefToImage("build/image", "3.2.1"), img)
		registry.Push(registry.RefToImage("build/image", "latest"), img)
		registry.Push(registry.RefToImage("build/image", "2.6.1"), img)

		repository := registry.RefToImage("build/image", "latest").Context()
		tags, err := ListTags(t.Context(), repository)
		require.NoError(t, err)

		slices.Sort(tags)
		require.Equal(t, []string{"2.6.1", "3", "3.2.1", "latest"}, tags)
	})

	t.Run("no tags", func(t *testing.T) {
		registry := fetchBldr.StartOCIRegistryServer(t)

		repository := registry.RefToImage("build/image", "latest").Context()
		tags, err := ListTags(t.Context(), repository)
		require.NoError(t, err)
		require.Empty(t, tags)
	})
}
