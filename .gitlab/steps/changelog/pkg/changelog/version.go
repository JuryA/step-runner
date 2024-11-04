package changelog

import "fmt"

type Version struct {
	Major   string
	Minor   string
	Patch   string
	Date    string
	Changes []string
}

func NewVersion(major, minor, patch, date string, changes []string) *Version {
	return &Version{Major: major, Minor: minor, Patch: patch, Date: date, Changes: changes}
}

func (v *Version) MajorMinorPatch() string {
	return fmt.Sprintf("%s.%s.%s", v.Major, v.Minor, v.Patch)
}
