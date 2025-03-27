package internal_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/dist-steps/oci/publish/internal"
	"gitlab.com/gitlab-org/dist-steps/oci/publish/internal/testutil/bldr"

	mainBldr "gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestArtifacts_ForPlatform(t *testing.T) {
	t.Run("filters by platform", func(t *testing.T) {
		generic := bldr.OCIArtifact(t).Generic().Build()
		linuxAmd64 := bldr.OCIArtifact(t).LinuxAMD64().Build()

		artifacts := internal.NewArtifacts(generic, linuxAmd64).ForPlatform(mainBldr.OCIPlatform.Generic)
		require.Len(t, artifacts, 1)
		require.Equal(t, generic, artifacts[0])
	})

	t.Run("returns zero artifacts when none match", func(t *testing.T) {
		generic := bldr.OCIArtifact(t).Generic().Build()

		artifacts := internal.NewArtifacts(generic).ForPlatform(mainBldr.OCIPlatform.LinuxARM64)
		require.Len(t, artifacts, 0)
	})
}

func TestArtifacts_Platforms(t *testing.T) {
	t.Run("returns unique platforms", func(t *testing.T) {
		linuxAmd64A := bldr.OCIArtifact(t).LinuxAMD64().Build()
		linuxArm64A := bldr.OCIArtifact(t).LinuxARM64().Build()
		linuxAmd64B := bldr.OCIArtifact(t).LinuxAMD64().Build()

		platforms := internal.NewArtifacts(linuxAmd64A, linuxArm64A, linuxAmd64B).Platforms()
		require.Len(t, platforms, 2)
		require.Equal(t, mainBldr.OCIPlatform.LinuxAMD64, platforms[0])
		require.Equal(t, mainBldr.OCIPlatform.LinuxARM64, platforms[1])
	})
}

func TestArtifacts_Add(t *testing.T) {
	a := bldr.OCIArtifact(t).Generic().Build()
	b := bldr.OCIArtifact(t).Generic().Build()
	c := bldr.OCIArtifact(t).LinuxARM64().Build()

	artifactsA := internal.NewArtifacts(a, b)
	artifactsB := internal.NewArtifacts(c)
	artifactsC := internal.NewArtifacts()

	artifacts := artifactsA.Add(artifactsB).Add(artifactsC)
	require.Len(t, artifacts, 3)
	require.Equal(t, a, artifacts[0])
	require.Equal(t, b, artifacts[1])
	require.Equal(t, c, artifacts[2])
}
