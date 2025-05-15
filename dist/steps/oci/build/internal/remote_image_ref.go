package internal

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

type RemoteImageRef struct {
	majorMinorPatch name.Reference
}

func NewRemoteImageRef(registry, repository, tag string) (*RemoteImageRef, error) {
	registry = strings.TrimSpace(registry)
	repository = strings.TrimSpace(repository)
	tag = strings.TrimSpace(tag)

	if registry == "" {
		return nil, errors.New("registry is required")
	}

	if repository == "" {
		return nil, errors.New("repository is required")
	}

	if tag == "" {
		return nil, errors.New("tag is required")
	}

	majorMinorPatch, err := name.ParseReference(fmt.Sprintf("%s:%s", path.Join(registry, repository), tag))
	if err != nil {
		return nil, fmt.Errorf("parsing image reference: %w", err)
	}

	return &RemoteImageRef{majorMinorPatch: majorMinorPatch}, nil
}

func (ri *RemoteImageRef) MajorMinorPatch() name.Reference {
	return ri.majorMinorPatch
}

func (ri *RemoteImageRef) String() string {
	return ri.majorMinorPatch.String()
}

func (ri *RemoteImageRef) Repository() name.Repository {
	return ri.majorMinorPatch.Context()
}
