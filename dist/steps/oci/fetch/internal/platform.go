package internal

import (
	"bytes"
	"log/slog"
	"strings"

	"github.com/containerd/platforms"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func FindManifestForPlatforms(findFor []platforms.Platform, manifests []v1.Descriptor) *v1.Descriptor {
	for _, platform := range findFor {
		slog.Debug("searching image index manifest for platform image", "platform", platform)
		matched := FindManifestForPlatform(platform, manifests)

		if matched != nil {
			slog.Debug("found image for platform", "image_digest", matched.Digest, "platform", platform)
			return matched
		}
	}

	return nil
}

func FindManifestForPlatform(findFor platforms.Platform, manifests []v1.Descriptor) *v1.Descriptor {
	var matched *v1.Descriptor
	var matchedPlatform platforms.Platform

	matcher := platforms.Only(findFor)

	for _, manifest := range manifests {
		platform := ConvertPlatformV1ToCtrd(manifest.Platform)

		if !matcher.Match(platform) {
			continue
		}

		if matched == nil || matcher.Less(platform, matchedPlatform) {
			matched = &manifest
			matchedPlatform = platform
		}
	}

	return matched
}

func ConvertPlatformV1ToCtrd(v1Platform *v1.Platform) platforms.Platform {
	return platforms.Platform{
		Architecture: v1Platform.Architecture,
		OS:           v1Platform.OS,
		OSVersion:    v1Platform.OSVersion,
		OSFeatures:   v1Platform.OSFeatures,
		Variant:      v1Platform.Variant,
	}
}

func DescribePlatforms(plats ...platforms.Platform) string {
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
