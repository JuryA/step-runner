package bldr

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
)

type OCIArtifactBuilder struct {
	dir      string
	platform *v1.Platform
}

func OCIArtifact(t *testing.T) *OCIArtifactBuilder {
	return &OCIArtifactBuilder{
		dir:      t.TempDir(),
		platform: OCIPlatform.LinuxARM64,
	}
}

func (bldr *OCIArtifactBuilder) Generic() *OCIArtifactBuilder {
	return bldr.WithPlatform(OCIPlatform.Generic)
}

func (bldr *OCIArtifactBuilder) LinuxAMD64() *OCIArtifactBuilder {
	return bldr.WithPlatform(OCIPlatform.LinuxAMD64)
}

func (bldr *OCIArtifactBuilder) LinuxARM64() *OCIArtifactBuilder {
	return bldr.WithPlatform(OCIPlatform.LinuxARM64)
}

func (bldr *OCIArtifactBuilder) WithPlatform(platform *v1.Platform) *OCIArtifactBuilder {
	bldr.platform = platform
	return bldr
}

func (bldr *OCIArtifactBuilder) WithDir(dir string) *OCIArtifactBuilder {
	bldr.dir = dir
	return bldr
}

func (bldr *OCIArtifactBuilder) Build() *oci.Artifact {
	return oci.NewArtifact(bldr.dir, bldr.platform)
}
