package pkg

import (
	"github.com/containerd/platforms"
	"github.com/google/go-containerregistry/pkg/v1"
)

type PlatformImage struct {
	Image    v1.Image
	Platform *v1.Platform
}

func (i PlatformImage) NormalizedPlatform() *v1.Platform {
	containerdPlatform := platforms.Platform{
		Architecture: i.Platform.Architecture,
		OS:           i.Platform.OS,
		OSVersion:    i.Platform.OSVersion,
		OSFeatures:   i.Platform.OSFeatures,
		Variant:      i.Platform.Variant,
	}

	normalized := platforms.Normalize(containerdPlatform)

	return &v1.Platform{
		Architecture: normalized.Architecture,
		OS:           normalized.OS,
		OSVersion:    normalized.OSVersion,
		OSFeatures:   normalized.OSFeatures,
		Variant:      normalized.Variant,
		Features:     i.Platform.Features,
	}
}
