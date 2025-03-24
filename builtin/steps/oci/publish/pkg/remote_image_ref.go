package pkg

import (
	"errors"
	"fmt"
	"log/slog"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

var majorMinorPatchRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)

type version struct {
	major   uint64
	minor   uint64
	patch   uint64
	release string
}

type RemoteImageRef struct {
	majorMinorPatch name.Reference
	version         *version
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

	tagParts := semVerRe.FindStringSubmatch(tag)

	if len(tagParts) != 5 {
		return nil, fmt.Errorf("tag does not conform to semantic versioning major.minor.patch[-release]: %s", tag)
	}

	major, err := strconv.ParseUint(tagParts[1], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("major version %s: %w", tagParts[1], err)
	}

	minor, err := strconv.ParseUint(tagParts[2], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("minor version: %s: %w", tagParts[2], err)
	}

	patch, err := strconv.ParseUint(tagParts[3], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("patch version: %s: %w", tagParts[3], err)
	}

	release := tagParts[4]
	majorMinorPatch, err := name.ParseReference(fmt.Sprintf("%s:%d.%d.%d%s", path.Join(registry, repository), major, minor, patch, release))
	if err != nil {
		return nil, fmt.Errorf("parsing image reference: %w", err)
	}

	ref := &RemoteImageRef{
		majorMinorPatch: majorMinorPatch,
		version: &version{
			major:   major,
			minor:   minor,
			patch:   patch,
			release: release,
		},
	}

	return ref, nil
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

// SemVerRefs determines which tags should be created for the published release.
// For example, releasing version 1.5.7 may require updating the MAJOR.MINOR version, 1.5, or the MAJOR version, 1.
// Major and minor tags are updated only when the latest version is released.
// Major and minor tags are not updated when a release candidate is released. (e.g. 1.4.5-rc1)
func (ri *RemoteImageRef) SemVerRefs(existingTags []string) ([]name.Reference, error) {
	refs := []name.Reference{ri.majorMinorPatch}

	if ri.version.release != "" {
		return refs, nil
	}

	tags := ri.parseTags(existingTags)
	namedRefs := []string{}

	if ri.isMostRecentMinor(tags) {
		namedRefs = append(namedRefs, fmt.Sprintf("%d.%d", ri.version.major, ri.version.minor))

		if ri.isMostRecentMajor(tags) {
			namedRefs = append(namedRefs, fmt.Sprintf("%d", ri.version.major))
		}
	}

	for _, namedRef := range namedRefs {
		ref, err := ri.buildRefForTag(namedRef)
		if err != nil {
			return nil, err
		}

		refs = append(refs, ref)
	}

	return refs, nil
}

func (ri *RemoteImageRef) parseTags(tags []string) []*version {
	versions := make([]*version, 0)

	for _, tag := range tags {
		tagVersion := ri.parseTag(tag)

		if tagVersion != nil {
			versions = append(versions, tagVersion)
		}
	}

	return versions
}

func (ri *RemoteImageRef) parseTag(tag string) *version {
	parts := majorMinorPatchRe.FindStringSubmatch(tag)

	if len(parts) != 4 {
		slog.Debug("ignoring published tag as it does not conform to semantic versioning major.minor.patch", "tag", tag)
		return nil
	}

	major, err := strconv.ParseUint(parts[1], 10, 0)
	if err != nil {
		slog.Debug("ignoring published tag as it does not conform to semantic versioning major.minor.patch", "tag", tag, "major", parts[1])
		return nil
	}

	minor, err := strconv.ParseUint(parts[2], 10, 0)
	if err != nil {
		slog.Debug("ignoring published tag as it does not conform to semantic versioning major.minor.patch", "tag", tag, "minor", parts[2])
		return nil
	}

	patch, err := strconv.ParseUint(parts[3], 10, 0)
	if err != nil {
		slog.Debug("ignoring published tag as it does not conform to semantic versioning major.minor.patch", "tag", tag, "patch", parts[3])
		return nil
	}

	return &version{
		major:   major,
		minor:   minor,
		patch:   patch,
		release: "",
	}
}

func (ri *RemoteImageRef) buildRefForTag(tag string) (name.Reference, error) {
	ref, err := name.ParseReference(fmt.Sprintf("%s:%s", ri.majorMinorPatch.Context().Name(), tag))
	if err != nil {
		return nil, fmt.Errorf("creating ref for tag %s: %w", tag, err)
	}

	return ref, nil
}

func (ri *RemoteImageRef) isMostRecentMinor(tags []*version) bool {
	for _, tag := range tags {
		if tag.major != ri.version.major || tag.minor != ri.version.minor {
			continue
		}

		if tag.patch > ri.version.patch {
			return false
		}
	}

	return true
}

func (ri *RemoteImageRef) isMostRecentMajor(tags []*version) bool {
	for _, tag := range tags {
		if tag.major != ri.version.major {
			continue
		}

		if tag.minor > ri.version.minor {
			return false
		}
	}

	return true
}
