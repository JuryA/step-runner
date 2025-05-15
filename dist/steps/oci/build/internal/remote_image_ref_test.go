package internal_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/dist/steps/oci/build/internal"
)

func TestNewRemoteImageRef(t *testing.T) {
	t.Run("registry", func(t *testing.T) {
		tests := []struct {
			name      string
			registry  string
			expect    string
			expectErr string
		}{
			{
				name:     "parses",
				registry: "registry.gitlab.com:5000",
				expect:   "registry.gitlab.com:5000",
			},
			{
				name:     "trims space",
				registry: "  registry.gitlab.com  ",
				expect:   "registry.gitlab.com",
			},
			{
				name:      "cannot be empty",
				registry:  "",
				expectErr: "registry is required",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				remoteImgRef, err := internal.NewRemoteImageRef(test.registry, "my-image", "1.0.0")

				if test.expectErr == "" {
					require.NoError(t, err)
					require.Equal(t, test.expect, remoteImgRef.Repository().RegistryStr())
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				}
			})
		}
	})

	t.Run("repository", func(t *testing.T) {
		tests := []struct {
			name       string
			repository string
			expect     string
			expectErr  string
		}{
			{
				name:       "parses",
				repository: "my_group/my_project/image",
				expect:     "my_group/my_project/image",
			},
			{
				name:       "trims space",
				repository: "  my_group/my_project/image  ",
				expect:     "my_group/my_project/image",
			},
			{
				name:       "cannot be empty",
				repository: "",
				expectErr:  "repository is required",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				remoteImgRef, err := internal.NewRemoteImageRef("reg.gl.com", test.repository, "1.0.0")

				if test.expectErr == "" {
					require.NoError(t, err)
					require.Equal(t, test.expect, remoteImgRef.Repository().RepositoryStr())
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				}
			})
		}
	})

	t.Run("tag", func(t *testing.T) {
		tests := []struct {
			name      string
			tag       string
			expect    string
			expectErr string
		}{
			{
				name:   "parses",
				tag:    "1.0.3",
				expect: "1.0.3",
			},
			{
				name:   "trims space",
				tag:    "  12.44.32  ",
				expect: "12.44.32",
			},
			{
				name:      "cannot be empty",
				tag:       "",
				expectErr: "tag is required",
			},
			{
				name:   "does not need to be semver",
				tag:    "pipeline-123343",
				expect: "pipeline-123343",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				remoteImgRef, err := internal.NewRemoteImageRef("reg.gl.com", "my-image", test.tag)

				if test.expectErr == "" {
					require.NoError(t, err)
					require.Contains(t, test.expect, remoteImgRef.MajorMinorPatch().Identifier())
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				}
			})
		}
	})
}
