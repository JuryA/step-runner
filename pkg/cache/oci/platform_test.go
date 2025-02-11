package oci_test

import (
	"testing"

	"github.com/containerd/platforms"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestPlatform_FindManifestForPlatforms(t *testing.T) {
	tests := []struct {
		name      string
		manifests []*v1.Platform
		findFor   []*v1.Platform
		expect    *v1.Platform
	}{
		{
			name:      "returns nil when not matched",
			manifests: []*v1.Platform{},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxAMD64},
			expect:    nil,
		},
		{
			name:      "finds by os and architecture",
			manifests: []*v1.Platform{bldr.OCIPlatform.LinuxAMD64},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxAMD64},
			expect:    bldr.OCIPlatform.LinuxAMD64,
		},
		{
			name:      "finds by os and architecture and variant",
			manifests: []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8},
			expect:    bldr.OCIPlatform.LinuxARM64v8,
		},
		{
			name:      "finds most specific when ordered by most specific",
			manifests: []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8, bldr.OCIPlatform.LinuxARM64v7},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8},
			expect:    bldr.OCIPlatform.LinuxARM64v8,
		},
		{
			name:      "finds most specific when ordered by least specific",
			manifests: []*v1.Platform{bldr.OCIPlatform.LinuxARM64v7, bldr.OCIPlatform.LinuxARM64v8},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8},
			expect:    bldr.OCIPlatform.LinuxARM64v8,
		},
		{
			name:      "arm v8 normalizes to arm64 (to match v7, v6 and v5)",
			manifests: []*v1.Platform{bldr.OCIPlatform.LinuxARM64},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8},
			expect:    bldr.OCIPlatform.LinuxARM64,
		},
		{
			name:      "arm matches arm v8",
			manifests: []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxARM64},
			expect:    bldr.OCIPlatform.LinuxARM64v8,
		},
		{
			name:      "arm v8 matches arm",
			manifests: []*v1.Platform{bldr.OCIPlatform.LinuxARM64},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxARM64v8},
			expect:    bldr.OCIPlatform.LinuxARM64,
		},
		{
			name:      "amd64 also matches 386",
			manifests: []*v1.Platform{bldr.OCIPlatform.Linux386},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxAMD64},
			expect:    bldr.OCIPlatform.Linux386,
		},
		{
			name:      "normalizes irregular names",
			manifests: []*v1.Platform{bldr.OCIPlatform.LinuxAMD64},
			findFor:   []*v1.Platform{{OS: "linux", Architecture: "x86_64"}},
			expect:    bldr.OCIPlatform.LinuxAMD64,
		},
		{
			name:      "returns platform matched first",
			manifests: []*v1.Platform{bldr.OCIPlatform.LinuxAMD64, bldr.OCIPlatform.Generic},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxAMD64, bldr.OCIPlatform.Generic},
			expect:    bldr.OCIPlatform.LinuxAMD64,
		},
		{
			name:      "falls back to other find when first not matched",
			manifests: []*v1.Platform{bldr.OCIPlatform.Generic},
			findFor:   []*v1.Platform{bldr.OCIPlatform.LinuxAMD64, bldr.OCIPlatform.Generic},
			expect:    bldr.OCIPlatform.Generic,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifests := make([]v1.Descriptor, len(test.manifests))
			for i, platform := range test.manifests {
				manifests[i] = v1.Descriptor{Platform: platform}
			}

			findFor := make([]platforms.Platform, len(test.findFor))
			for i, platform := range test.findFor {
				findFor[i] = oci.ConvertPlatformV1ToCtrd(platform)
			}

			matched := oci.FindManifestForPlatforms(findFor, manifests)
			if test.expect == nil {
				require.Nil(t, matched)
			} else {
				require.NotNil(t, matched)
				require.Equal(t, test.expect, (*matched).Platform)
			}
		})
	}
}
