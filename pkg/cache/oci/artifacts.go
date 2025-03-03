package oci

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

var PlatformGeneric = &v1.Platform{OS: "generic", Architecture: "generic"}

type Artifacts struct {
	values []*Artifact
}

func NewArtifacts(artifacts ...*Artifact) *Artifacts {
	return &Artifacts{
		values: artifacts,
	}
}

func (a *Artifacts) ForPlatform(platform *v1.Platform) *Artifacts {
	values := make([]*Artifact, 0)

	for _, artifact := range a.values {
		if artifact.Platform.Equals(*platform) {
			values = append(values, artifact)
		}
	}

	return NewArtifacts(values...)
}

func (a *Artifacts) Generic() *Artifacts {
	return a.ForPlatform(PlatformGeneric)
}

// Platforms returns a unique list of platforms represented by the artifacts.
// The generic platform is excluded from the result set.
func (a *Artifacts) Platforms() []*v1.Platform {
	unique := make([]*v1.Platform, 0)

	// O(n^2) approach due to a lack reliable platform hash function
	for _, artifact := range a.values {
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

func (a *Artifacts) Add(artifacts *Artifacts) *Artifacts {
	combined := make([]*Artifact, 0, len(a.values)+len(artifacts.values))
	combined = append(combined, a.values...)
	combined = append(combined, artifacts.values...)
	return NewArtifacts(combined...)
}

func (a *Artifacts) Values() []*Artifact {
	return a.values
}

func (a *Artifacts) Len() int {
	return len(a.values)
}

func (a *Artifacts) Nth(i int) *Artifact {
	return a.values[i]
}
