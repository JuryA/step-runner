package internal

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

var semVerRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(-.*)?$`)

type RemoteImageRef struct {
	ref     name.Reference
	version *Version
}

func ParseRemoteImageRef(registry, repository, version string) (*RemoteImageRef, error) {
	registry = strings.TrimSpace(registry)
	repository = strings.TrimSpace(repository)
	version = strings.TrimSpace(version)

	if registry == "" {
		return nil, errors.New("registry is required")
	}

	if repository == "" {
		return nil, errors.New("repository is required")
	}

	if version == "" {
		return nil, errors.New("version is required")
	}

	versionParts := semVerRe.FindStringSubmatch(version)

	if len(versionParts) != 5 {
		return nil, fmt.Errorf("version does not conform to semantic versioning major.minor.patch[-release]: %s", version)
	}

	major, err := strconv.ParseUint(versionParts[1], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("major version %s: %w", versionParts[1], err)
	}

	minor, err := strconv.ParseUint(versionParts[2], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("minor version: %s: %w", versionParts[2], err)
	}

	patch, err := strconv.ParseUint(versionParts[3], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("patch version: %s: %w", versionParts[3], err)
	}

	release := versionParts[4]
	ref, err := name.ParseReference(fmt.Sprintf("%s:%d.%d.%d%s", path.Join(registry, repository), major, minor, patch, release))
	if err != nil {
		return nil, fmt.Errorf("parsing image reference: %w", err)
	}

	return NewRemoteImageRef(ref, NewVersion(major, minor, patch, release)), nil
}

func NewRemoteImageRef(ref name.Reference, version *Version) *RemoteImageRef {
	return &RemoteImageRef{
		ref:     ref,
		version: version,
	}
}

func (ri *RemoteImageRef) MajorMinorPatch() name.Reference {
	return ri.ref
}

func (ri *RemoteImageRef) String() string {
	return ri.ref.String()
}

func (ri *RemoteImageRef) Repository() name.Repository {
	return ri.ref.Context()
}
