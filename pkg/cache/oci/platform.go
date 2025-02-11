package oci

import (
	"bytes"
	"strings"

	"github.com/containerd/platforms"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func ConvertV1PlatformToContainerdPlatform(v1Platform *v1.Platform) platforms.Platform {
	return platforms.Platform{
		Architecture: v1Platform.Architecture,
		OS:           v1Platform.OS,
		OSVersion:    v1Platform.OSVersion,
		OSFeatures:   v1Platform.OSFeatures,
		Variant:      v1Platform.Variant,
	}
}

func DescribePlatforms(plats []platforms.Platform) string {
	descriptions := []string{}

	for _, platform := range plats {
		descriptions = append(descriptions, DescribePlatform(platform))
	}

	return strings.Join(descriptions, " or ")
}

func DescribePlatform(platform platforms.Platform) string {
	description := bytes.NewBufferString(platform.OS)

	if platform.Architecture != "" {
		description.WriteString("/")
		description.WriteString(platform.Architecture)
	}

	if platform.Variant != "" {
		description.WriteString("/")
		description.WriteString(platform.Variant)
	}

	return description.String()
}
