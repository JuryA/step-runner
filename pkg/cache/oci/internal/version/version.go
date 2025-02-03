package version

import (
	"cmp"
	"errors"
	"slices"
	"strconv"
	"strings"
)

var (
	ErrInvalidSemanticVersion = errors.New("invalid semantic version")
	ErrInvalidPrerelease      = errors.New("invalid prerelease")
)

type Version struct {
	Major int64
	Minor int64
	Patch int64

	Prerelease []string
}

func New(input string) (Version, error) {
	return parse(input, false)
}

func (v Version) String() string {
	var sb strings.Builder

	sb.WriteString(strconv.FormatInt(v.Major, 10))

	sb.WriteByte('.')
	sb.WriteString(strconv.FormatInt(v.Minor, 10))

	sb.WriteByte('.')
	sb.WriteString(strconv.FormatInt(v.Patch, 10))

	if len(v.Prerelease) > 0 {
		sb.WriteByte('-')
		sb.WriteString(v.Prerelease[0])
		for _, part := range v.Prerelease[1:] {
			sb.WriteByte('.')
			sb.WriteString(part)
		}
	}

	return sb.String()
}

func (v Version) LessThan(ver Version) bool {
	return v.Compare(ver) < 0
}

func (v Version) GreaterThan(ver Version) bool {
	return v.Compare(ver) > 0
}

func (v Version) Equal(ver Version) bool {
	return v.Compare(ver) == 0
}

func (v Version) Compare(ver Version) int {
	return Compare(v, ver)
}

func Compare(a, b Version) int {
	switch {
	case a.Major == Any || a.Major == Latest: // constraint-based version
		break

	case a.Major < b.Major:
		return -1
	case a.Major > b.Major:
		return 1

	case a.Minor == Any: // constraint-based version
		break

	case a.Minor < b.Minor:
		return -1
	case a.Minor > b.Minor:
		return 1

	case a.Patch == Any: // constraint-based version
		break

	case a.Patch < b.Patch:
		return -1
	case a.Patch > b.Patch:
		return 1
	}

	if a.Prerelease != nil && b.Prerelease == nil {
		return -1
	}

	if a.Prerelease == nil && b.Prerelease != nil {
		return 1
	}

	return slices.CompareFunc(a.Prerelease, b.Prerelease, func(a, b string) int {
		if a == b {
			return 0
		}

		aint, aerr := strconv.ParseInt(a, 10, 64)
		bint, berr := strconv.ParseInt(b, 10, 64)

		switch {
		case aerr == nil && berr == nil:
			return cmp.Compare(aint, bint)

		case aerr == nil:
			return -1

		case berr == nil:
			return 1
		}

		return cmp.Compare(a, b)
	})
}
