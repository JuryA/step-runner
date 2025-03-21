package pkg_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/builtin/steps/oci/publish/pkg"
)

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
			expectPublishTags: []string{"3.5.1", "3.5", "3"},
		},
		{
			name:              "publishes major and minor when no tags exist",
			existingTags:      []string{},
			publish:           "3.5.1",
			expectPublishTags: []string{"3.5.1", "3.5", "3"},
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
			expectPublishTags: []string{"5.7.1", "5.7", "5"},
		},
		{
			name:              "publised release candidates are ignored",
			existingTags:      []string{"5.7.1-rc1"},
			publish:           "5.7.0",
			expectPublishTags: []string{"5.7.0", "5.7", "5"},
		},
	}

	semVerRe := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(-.*)?$`)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			imgRef, err := name.ParseReference("registry.gitlab.com/my-project/image:" + test.publish)
			require.NoError(t, err)

			tagParts := semVerRe.FindStringSubmatch(test.publish)
			require.Len(t, tagParts, 5)

			major, err := strconv.ParseUint(tagParts[1], 10, 0)
			require.NoError(t, err)

			minor, err := strconv.ParseUint(tagParts[2], 10, 0)
			require.NoError(t, err)

			patch, err := strconv.ParseUint(tagParts[3], 10, 0)
			require.NoError(t, err)
			release := tagParts[4]

			remoteImgRef := pkg.NewRemoteImageRef(imgRef, major, minor, patch, release)
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
