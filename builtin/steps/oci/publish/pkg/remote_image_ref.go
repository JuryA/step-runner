package pkg

import (
	"github.com/google/go-containerregistry/pkg/name"
)

type RemoteImageRef struct {
	majorMinorPatch name.Reference
	major           uint64
	minor           uint64
	patch           uint64
	release         string
}

func NewRemoteImageRef(majorMinorPatch name.Reference, major, minor, patch uint64, release string) *RemoteImageRef {
	return &RemoteImageRef{
		majorMinorPatch: majorMinorPatch,
		major:           major,
		minor:           minor,
		patch:           patch,
		release:         release,
	}
}

func (ri *RemoteImageRef) MajorMinorPatch() name.Reference {
	return ri.majorMinorPatch
}

func (ri *RemoteImageRef) String() string {
	return ri.majorMinorPatch.String()
}
