package bldr

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/build/internal"

	mainBldr "gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

type OCIArtifactBuilder struct {
	from     string
	to       string
	platform *v1.Platform
}

func OCIArtifact(t *testing.T) *OCIArtifactBuilder {
	return &OCIArtifactBuilder{
		from:     t.TempDir(),
		to:       "/my_step",
		platform: mainBldr.OCIPlatform.LinuxARM64,
	}
}

func (bldr *OCIArtifactBuilder) Generic() *OCIArtifactBuilder {
	return bldr.WithPlatform(mainBldr.OCIPlatform.Generic)
}

func (bldr *OCIArtifactBuilder) LinuxAMD64() *OCIArtifactBuilder {
	return bldr.WithPlatform(mainBldr.OCIPlatform.LinuxAMD64)
}

func (bldr *OCIArtifactBuilder) LinuxARM64() *OCIArtifactBuilder {
	return bldr.WithPlatform(mainBldr.OCIPlatform.LinuxARM64)
}

func (bldr *OCIArtifactBuilder) WithPlatform(platform *v1.Platform) *OCIArtifactBuilder {
	bldr.platform = platform
	return bldr
}

func (bldr *OCIArtifactBuilder) WithFrom(from string) *OCIArtifactBuilder {
	bldr.from = from
	return bldr
}

func (bldr *OCIArtifactBuilder) WithTo(to string) *OCIArtifactBuilder {
	bldr.to = to
	return bldr
}

func (bldr *OCIArtifactBuilder) Build() *internal.Artifact {
	return internal.NewArtifact(bldr.platform, bldr.from, bldr.to)
}

func (bldr *OCIArtifactBuilder) BuildArtifacts() internal.Artifacts {
	return internal.NewArtifacts(bldr.Build())
}
