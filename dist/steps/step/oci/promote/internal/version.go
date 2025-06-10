package internal

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
)

var majorMinorPatchRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)

func ParseSemanticVersions(tags []string) Versions {
	versions := make([]*Version, 0)

	for _, tag := range tags {
		version, err := ParseSemanticVersion(tag)
		if err != nil {
			slog.Debug("ignoring published tag", "err", err.Error())
			continue
		}

		versions = append(versions, version)
	}

	return versions
}

func ParseSemanticVersion(tag string) (*Version, error) {
	parts := majorMinorPatchRe.FindStringSubmatch(tag)

	if len(parts) != 4 {
		return nil, fmt.Errorf("parse tag %s: does not conform to semver major.minor.patch", tag)
	}

	major, err := strconv.ParseUint(parts[1], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("parse tag %s: does not conform to semver major.minor.patch", tag)
	}

	minor, err := strconv.ParseUint(parts[2], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("parse tag %s: does not conform to semver major.minor.patch", tag)
	}

	patch, err := strconv.ParseUint(parts[3], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("parse tag %s: does not conform to semver major.minor.patch", tag)
	}

	return NewVersion(major, minor, patch, ""), nil
}

type Versions []*Version

type Version struct {
	major   uint64
	minor   uint64
	patch   uint64
	release string
}

func NewVersion(major, minor, patch uint64, release string) *Version {
	return &Version{
		major:   major,
		minor:   minor,
		patch:   patch,
		release: release,
	}
}

func (v *Version) IsReleaseCandidate() bool {
	return v.release != ""
}

func (v *Version) TagsToUpdate(existing Versions) []string {
	newTags := []string{fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)}

	if v.isMostRecentPatchForMajorMinor(existing) {
		newTags = append(newTags, fmt.Sprintf("%d.%d", v.major, v.minor))

		if v.isMostRecentMinorForMajor(existing) {
			newTags = append(newTags, fmt.Sprintf("%d", v.major))

			if v.isMostRecentMajor(existing) {
				newTags = append(newTags, "latest")
			}
		}
	}

	return newTags
}

func (v *Version) isMostRecentPatchForMajorMinor(existing Versions) bool {
	for _, e := range existing {
		if e.major != v.major || e.minor != v.minor {
			continue
		}

		if e.patch > v.patch {
			return false
		}
	}

	return true
}

func (v *Version) isMostRecentMinorForMajor(existing Versions) bool {
	for _, e := range existing {
		if e.major != v.major {
			continue
		}

		if e.minor > v.minor {
			return false
		}
	}

	return true
}

func (v *Version) isMostRecentMajor(existing []*Version) bool {
	for _, e := range existing {
		if e.major > v.major {
			return false
		}
	}

	return true
}
