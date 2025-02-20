package oci_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestArtifacts_ForPlatform(t *testing.T) {
	t.Run("filters by platform", func(t *testing.T) {
		generic := oci.NewArtifact("/common", bldr.OCIPlatform.Generic)
		linuxAmd64 := oci.NewArtifact("/linux/amd64", bldr.OCIPlatform.LinuxAMD64)

		artifacts := oci.NewArtifacts(generic, linuxAmd64).ForPlatform(bldr.OCIPlatform.Generic)
		require.Len(t, artifacts.Values(), 1)
		require.Equal(t, generic, artifacts.Values()[0])
	})

	t.Run("returns zero artifacts when none match", func(t *testing.T) {
		generic := oci.NewArtifact("/common", bldr.OCIPlatform.Generic)

		artifacts := oci.NewArtifacts(generic).ForPlatform(bldr.OCIPlatform.LinuxARM64)
		require.Len(t, artifacts.Values(), 0)
	})
}

func TestArtifacts_Platforms(t *testing.T) {
	t.Run("returns unique platforms", func(t *testing.T) {
		linuxAmd64A := oci.NewArtifact("/linux/amd64", bldr.OCIPlatform.LinuxAMD64)
		linuxArm64A := oci.NewArtifact("/linux/arm64/a", bldr.OCIPlatform.LinuxARM64)
		linuxAmd64B := oci.NewArtifact("/linux/arm64/b", bldr.OCIPlatform.LinuxARM64)

		platforms := oci.NewArtifacts(linuxAmd64A, linuxArm64A, linuxAmd64B).Platforms()
		require.Len(t, platforms, 2)
		require.Equal(t, bldr.OCIPlatform.LinuxAMD64, platforms[0])
		require.Equal(t, bldr.OCIPlatform.LinuxARM64, platforms[1])
	})

	t.Run("excludes generic as a platform", func(t *testing.T) {
		generic := oci.NewArtifact("/common", bldr.OCIPlatform.Generic)
		linuxAmd64 := oci.NewArtifact("/linux/amd64", bldr.OCIPlatform.LinuxAMD64)

		platforms := oci.NewArtifacts(generic, linuxAmd64).Platforms()
		require.Len(t, platforms, 1)
		require.Equal(t, bldr.OCIPlatform.LinuxAMD64, platforms[0])
	})
}

func TestArtifacts_Generic(t *testing.T) {
	genericA := oci.NewArtifact("/common/a", bldr.OCIPlatform.Generic)
	genericB := oci.NewArtifact("/common/b", bldr.OCIPlatform.Generic)
	linuxArm64 := oci.NewArtifact("/linux/arm64", bldr.OCIPlatform.LinuxARM64)

	artifacts := oci.NewArtifacts(genericA, genericB, linuxArm64).Generic()
	require.Len(t, artifacts.Values(), 2)
	require.Equal(t, genericA, artifacts.Values()[0])
	require.Equal(t, genericB, artifacts.Values()[1])
}

func TestArtifacts_Add(t *testing.T) {
	a := oci.NewArtifact("/common/a", bldr.OCIPlatform.Generic)
	b := oci.NewArtifact("/common/b", bldr.OCIPlatform.Generic)
	c := oci.NewArtifact("/linux/arm64", bldr.OCIPlatform.LinuxARM64)

	artifactsA := oci.NewArtifacts(a, b)
	artifactsB := oci.NewArtifacts(c)
	artifactsC := oci.NewArtifacts()

	artifacts := artifactsA.Add(artifactsB).Add(artifactsC)
	require.Len(t, artifacts.Values(), 3)
	require.Equal(t, a, artifacts.Values()[0])
	require.Equal(t, b, artifacts.Values()[1])
	require.Equal(t, c, artifacts.Values()[2])
}
