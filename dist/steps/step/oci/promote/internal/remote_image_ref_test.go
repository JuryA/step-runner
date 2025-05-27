package internal_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/internal"
)

func TestParseRemoteImageRef(t *testing.T) {
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
				remoteImgRef, err := internal.ParseRemoteImageRef(test.registry, "my-image", "1.0.0")

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
				remoteImgRef, err := internal.ParseRemoteImageRef("reg.gl.com", test.repository, "1.0.0")

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

	t.Run("version", func(t *testing.T) {
		tests := []struct {
			name      string
			version   string
			expect    string
			expectErr string
		}{
			{
				name:    "parses",
				version: "1.0.3",
				expect:  "1.0.3",
			},
			{
				name:    "trims space",
				version: "  12.44.32  ",
				expect:  "12.44.32",
			},
			{
				name:      "cannot be empty",
				version:   "",
				expectErr: "version is required",
			},
			{
				name:      "version must be semver including patch",
				version:   "latest",
				expectErr: `version does not conform to semantic versioning major.minor.patch[-release]: latest`,
			},
			{
				name:      "version cannot be major and minor only",
				version:   "2.0",
				expectErr: `version does not conform to semantic versioning major.minor.patch[-release]: 2.0`,
			},
			{
				name:    "version can include release candidate",
				version: "2.0.0-rc1",
				expect:  "2.0.0-rc1",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				remoteImgRef, err := internal.ParseRemoteImageRef("reg.gl.com", "my-image", test.version)

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
