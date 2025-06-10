package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSemanticVersion(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		expect    *Version
		expectErr bool
	}{
		{name: "major.minor.patch", version: "2.6.2", expect: NewVersion(2, 6, 2, "")},
		{name: "major", version: "2", expectErr: true},
		{name: "major.minor", version: "1.6", expectErr: true},
		{name: "major.minor.patch-release", version: "1.6.3-rc1", expectErr: true},
		{name: "not semver", version: "edge", expectErr: true},
		{name: "empty string", version: "", expectErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version, err := ParseSemanticVersion(test.version)
			if test.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expect, version)
			}
		})
	}
}

func TestVersion_TagsToUpdate(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		existing      []string
		expectNewTags []string
	}{
		{
			name:          "publishes major and minor when publishing latest tag",
			version:       "3.5.1",
			existing:      []string{"3", "3.5", "3.5.0"},
			expectNewTags: []string{"3.5.1", "3.5", "3", "latest"},
		},
		{
			name:          "publishes major and minor when no tags exist",
			version:       "3.5.1",
			existing:      []string{},
			expectNewTags: []string{"3.5.1", "3.5", "3", "latest"},
		},
		{
			name:          "publishes major when updating old minor",
			version:       "3.5.1",
			existing:      []string{"3", "3.6", "3.6.0", "3.5", "3.5.0"},
			expectNewTags: []string{"3.5.1", "3.5"},
		},
		{
			name:          "publishes old major and minor when updating old major",
			version:       "2.1.2",
			existing:      []string{"3", "3.5", "3.5.0", "2", "2.1", "2.1.0", "2.1.1"},
			expectNewTags: []string{"2.1.2", "2.1", "2"},
		},
		{
			name:          "don't update major or minor when when updating not latest",
			version:       "2.1.2",
			existing:      []string{"3", "3.5", "3.5.0", "2", "2.2", "2.2.0", "2.1", "2.1.1", "2.1.3"},
			expectNewTags: []string{"2.1.2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version, err := ParseSemanticVersion(test.version)
			require.NoError(t, err)

			newTags := version.TagsToUpdate(ParseSemanticVersions(test.existing))
			require.Equal(t, test.expectNewTags, newTags)
		})
	}
}
