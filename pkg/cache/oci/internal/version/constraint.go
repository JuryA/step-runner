package version

import (
	"slices"
	"strconv"
	"strings"
)

type Constraint Version

func NewConstraint(input string) (Constraint, error) {
	v, err := parse(input, true)

	return Constraint(v), err
}

func (c Constraint) String() string {
	if c.Major == Latest {
		return "latest"
	}

	if c.Major == Any {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(strconv.FormatInt(c.Major, 10))

	if c.Minor > Any {
		sb.WriteByte('.')
		sb.WriteString(strconv.FormatInt(c.Minor, 10))
	}

	if c.Patch > Any {
		sb.WriteByte('.')
		sb.WriteString(strconv.FormatInt(c.Patch, 10))
	}

	if len(c.Prerelease) > 0 {
		sb.WriteByte('-')
		sb.WriteString(c.Prerelease[0])
		for _, part := range c.Prerelease[1:] {
			sb.WriteByte('.')
			sb.WriteString(part)
		}
	}

	return sb.String()
}

func (c Constraint) Match(versions []Version) []Version {
	var results []Version

	for _, v := range versions {
		if Version(c).Equal(v) {
			results = append(results, v)
		}
	}

	slices.SortFunc(results, Compare)

	if c.Major == Latest && len(results) > 0 {
		return []Version{results[len(results)-1]}
	}

	return results
}

func (c Constraint) IsVersion() bool {
	return c.Major != Latest && c.Major != Any && c.Minor != Any && c.Patch != Any
}
