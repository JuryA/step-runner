package bldr

import (
	"testing"

	"github.com/containerd/platforms"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type OCIImageIndexBuilder struct {
	t     *testing.T
	index v1.ImageIndex
}

func OCIImageIndex(t *testing.T) *OCIImageIndexBuilder {
	return &OCIImageIndexBuilder{
		t:     t,
		index: empty.Index,
	}
}

func (b *OCIImageIndexBuilder) WithPlatformImage(platform *v1.Platform, image v1.Image) *OCIImageIndexBuilder {
	b.index = mutate.AppendManifests(b.index, mutate.IndexAddendum{
		Add: image,
		Descriptor: v1.Descriptor{
			MediaType: types.OCIManifestSchema1,
			Platform:  platform,
		},
	})

	return b
}

func (b *OCIImageIndexBuilder) WithImageForThisPlatform(img v1.Image) *OCIImageIndexBuilder {
	thisPlatform := platforms.DefaultSpec()
	v1Platform := &v1.Platform{
		OS:           thisPlatform.OS,
		Architecture: thisPlatform.Architecture,
		Variant:      thisPlatform.Variant,
		OSVersion:    thisPlatform.OSVersion,
		OSFeatures:   thisPlatform.OSFeatures,
		Features:     nil,
	}

	return b.WithPlatformImage(v1Platform, img)
}

func (b *OCIImageIndexBuilder) Build() v1.ImageIndex {
	return b.index
}

var OCIPlatform = struct {
	LinuxAMD64   *v1.Platform
	Linux386     *v1.Platform
	LinuxARM64v8 *v1.Platform
	LinuxARM64v7 *v1.Platform
	LinuxARM64   *v1.Platform
	WindowsAMD64 *v1.Platform
	Generic      *v1.Platform
}{
	LinuxAMD64:   &v1.Platform{OS: "linux", Architecture: "amd64"},
	Linux386:     &v1.Platform{OS: "linux", Architecture: "386"},
	LinuxARM64v8: &v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"},
	LinuxARM64v7: &v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v7"},
	LinuxARM64:   &v1.Platform{OS: "linux", Architecture: "arm64"},
	WindowsAMD64: &v1.Platform{OS: "windows", Architecture: "amd64", OSVersion: "10.0.26100.2894"},
	Generic:      &v1.Platform{OS: "generic"},
}
