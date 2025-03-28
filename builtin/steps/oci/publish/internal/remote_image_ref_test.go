package internal_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-builtins/oci/publish/internal"
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
				name:      "tag must be semver including patch",
				tag:       "latest",
				expectErr: `tag does not conform to semantic versioning major.minor.patch[-release]: latest`,
			},
			{
				name:      "tag cannot be major and minor only",
				tag:       "2.0",
				expectErr: `tag does not conform to semantic versioning major.minor.patch[-release]: 2.0`,
			},
			{
				name:   "tag can include release candidate",
				tag:    "2.0.0-rc1",
				expect: "2.0.0-rc1",
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

func TestRemoteImageRef_SemVerRefs(t *testing.T) {
	tests := []struct {
		name              string
		existingTags      []string
		publish           string
		expectPublishTags []string
	}{
		{
			name:              "publishes major and minor when publishing latest tag",
			existingTags:      []string{"3", "3.5", "3.5.0"},
			publish:           "3.5.1",
			expectPublishTags: []string{"3.5.1", "3.5", "3", "latest"},
		},
		{
			name:              "publishes major and minor when no tags exist",
			existingTags:      []string{},
			publish:           "3.5.1",
			expectPublishTags: []string{"3.5.1", "3.5", "3", "latest"},
		},
		{
			name:              "publishes major when updating old minor",
			existingTags:      []string{"3", "3.6", "3.6.0", "3.5", "3.5.0"},
			publish:           "3.5.1",
			expectPublishTags: []string{"3.5.1", "3.5"},
		},
		{
			name:              "don't update major or minor when publishing a release candidate",
			existingTags:      []string{"3", "3.5", "3.5.0"},
			publish:           "3.5.1-rc1",
			expectPublishTags: []string{"3.5.1-rc1"},
		},
		{
			name:              "publishes old major and minor when updating old major",
			existingTags:      []string{"3", "3.5", "3.5.0", "2", "2.1", "2.1.0", "2.1.1"},
			publish:           "2.1.2",
			expectPublishTags: []string{"2.1.2", "2.1", "2"},
		},
		{
			name:              "don't update major or minor when when updating not latest",
			existingTags:      []string{"3", "3.5", "3.5.0", "2", "2.2", "2.2.0", "2.1", "2.1.1", "2.1.3"},
			publish:           "2.1.2",
			expectPublishTags: []string{"2.1.2"},
		},
		{
			name:              "malformed existing tags are ignored",
			existingTags:      []string{"5.7ish"},
			publish:           "5.7.1",
			expectPublishTags: []string{"5.7.1", "5.7", "5", "latest"},
		},
		{
			name:              "published release candidates are ignored",
			existingTags:      []string{"5.7.1-rc1"},
			publish:           "5.7.0",
			expectPublishTags: []string{"5.7.0", "5.7", "5", "latest"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			remoteImgRef, err := internal.NewRemoteImageRef("registry.gitlab.com", "project/image", test.publish)
			require.NoError(t, err)

			refs, err := remoteImgRef.SemVerRefs(test.existingTags)
			require.NoError(t, err)

			tags := make([]string, 0, len(refs))
			for _, ref := range refs {
				tags = append(tags, ref.Identifier())
			}

			require.Equal(t, test.expectPublishTags, tags)
		})
	}
}
