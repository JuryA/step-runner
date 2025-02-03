package version

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	Any    = -1
	Latest = -2
)

func parse(input string, constraint bool) (Version, error) {
	parts := strings.SplitN(input, ".", 3)
	if !constraint && len(parts) != 3 {
		return Version{}, ErrInvalidSemanticVersion
	}

	digits := make([]int64, 3)
	if constraint {
		if input == "" {
			return Version{Any, Any, Any, nil}, nil
		}
		if input == "latest" {
			digits[0] = Latest
			parts = nil
		}
	}
	digits[1] = Any
	digits[2] = Any

	var prerelease []string
	if len(parts) == 3 {
		minor, pre, ok := strings.Cut(parts[2], "-")
		parts[2] = minor

		// validate prerelease
		if ok {
			prerelease = strings.Split(pre, ".")
			for _, part := range prerelease {
				if part == "" {
					return Version{}, ErrInvalidPrerelease
				}
				if strings.ContainsFunc(part, func(r rune) bool {
					return !(('0' <= r && r <= '9') || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') || r == '-')
				}) {
					return Version{}, ErrInvalidPrerelease
				}
			}
		}
	}

	for idx := range parts {
		digit, err := strconv.ParseInt(parts[idx], 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("%w: part %q invalid: %w", ErrInvalidSemanticVersion, parts[idx], err)
		}

		digits[idx] = digit
	}

	if !constraint && (digits[0] < 0 || digits[1] < 0 || digits[2] < 0) {
		return Version{}, ErrInvalidSemanticVersion
	}

	return Version{digits[0], digits[1], digits[2], prerelease}, nil
}
