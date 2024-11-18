package changelog

import "fmt"

type Version struct {
	major   string
	minor   string
	patch   string
	changes []string
}

func NewVersion(major, minor, patch string, changes []string) *Version {
	return &Version{major: major, minor: minor, patch: patch, changes: changes}
}

func (v *Version) Tag() string {
	return fmt.Sprintf("v%s", v.MajorMinorPatch())
}

func (v *Version) Major() string {
	return v.major
}

func (v *Version) MajorMinor() string {
	return fmt.Sprintf("%s.%s", v.major, v.minor)
}

func (v *Version) MajorMinorPatch() string {
	return fmt.Sprintf("%s.%s.%s", v.major, v.minor, v.patch)
}

func (v *Version) Changes() []string {
	return v.changes
}
