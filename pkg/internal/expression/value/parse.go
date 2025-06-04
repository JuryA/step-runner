package value

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errStringNotTerminating = errors.New("string not terminating")
	errInvalidEscape        = errors.New("invalid escape")
	errInvalidNumber        = errors.New("invalid number")
)

// ParseNumber parses a string number and returns a Value with of kind Number.
func ParseNumber(v string) (Value, error) {
	f, ok := getBigFloat().SetString(v)
	if !ok {
		return null, errInvalidNumber
	}

	return Value{kind: NumberKind, v: f}, nil
}

// ParseString parses a literal string. It's similar to strconv.Unquote.
func ParseString(v string) (Value, error) {
	if len(v) < 2 {
		return null, errStringNotTerminating
	}

	mode := v[0]
	if v[len(v)-1] != mode || mode != '"' && mode != '\'' {
		return null, errStringNotTerminating
	}

	quote := false
	var sb strings.Builder
	for _, p := range v[1 : len(v)-1] {
		switch {
		case quote && mode == '"':
			switch p {
			case 'a':
				sb.WriteByte('\a')
			case 'b':
				sb.WriteByte('\b')
			case 'f':
				sb.WriteByte('\f')
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case 'v':
				sb.WriteByte('\v')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '$':
				sb.WriteByte('$')
			default:
				return null, fmt.Errorf("%w: \\%v", errInvalidEscape, string(p))
			}
			quote = false

		case quote && mode == '\'':
			switch p {
			case '\\':
				sb.WriteByte('\\')
			case '\'':
				sb.WriteByte('\'')
			case '$':
				sb.WriteByte('$')
			default:
				return null, fmt.Errorf("%w: \\%v", errInvalidEscape, string(p))
			}
			quote = false

		case p == '\\':
			quote = true

		default:
			sb.WriteRune(p)
		}
	}

	return String(sb.String()), nil
}
