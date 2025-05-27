package internal

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
