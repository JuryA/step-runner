package bldr

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

var OCIPlatformLinuxAMD64 = &v1.Platform{OS: "linux", Architecture: "amd64"}
var OCIPlatformLinuxARM64v8 = &v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"}

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

func (b *OCIImageIndexBuilder) Build() v1.ImageIndex {
	return b.index
}
