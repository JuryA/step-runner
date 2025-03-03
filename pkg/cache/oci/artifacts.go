package oci

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

var PlatformGeneric = &v1.Platform{OS: "generic", Architecture: "generic"}

type Artifacts []*Artifact

func NewArtifacts(artifacts ...*Artifact) Artifacts {
	return artifacts
}

func (a Artifacts) ForPlatform(platform *v1.Platform) Artifacts {
	values := make([]*Artifact, 0)

	for _, artifact := range a {
		if artifact.Platform.Equals(*platform) {
			values = append(values, artifact)
		}
	}

	return values
}

func (a Artifacts) Generic() Artifacts {
	return a.ForPlatform(PlatformGeneric)
}

// Platforms returns a unique list of platforms represented by the artifacts.
// The generic platform is excluded from the result set.
func (a Artifacts) Platforms() []*v1.Platform {
	unique := make([]*v1.Platform, 0)

	// O(n^2) approach due to a lack reliable platform hash function
	for _, artifact := range a {
		seen := false

		for _, platform := range unique {
			if artifact.Platform.Equals(*platform) {
				seen = true
				continue
			}
		}

		if !seen && !artifact.Platform.Equals(*PlatformGeneric) {
			unique = append(unique, artifact.Platform)
		}
	}

	return unique
}

func (a Artifacts) Add(artifacts Artifacts) Artifacts {
	combined := make([]*Artifact, 0, len(a)+len(artifacts))
	combined = append(combined, a...)
	combined = append(combined, artifacts...)
	return NewArtifacts(combined...)
}
