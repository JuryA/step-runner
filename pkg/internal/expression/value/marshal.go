package value

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

func (v Value) MarshalJSON() ([]byte, error) {
	var formatter func(v Value, depth int, indent bool) string

	formatter = func(v Value, depth int, indent bool) string {
		return pretty(v, depth, indent, formatter)
	}

	return []byte(pretty(v, 0, true, func(v Value, depth int, indent bool) string {
		return formatter(v, depth, indent)
	})), nil
}

func pretty(v Value, depth int, indent bool, fn func(v Value, depth int, indent bool) string) string {
	var prefix string
	var nextPrefix string
	if indent {
		prefix = strings.Repeat("\t", depth)
		nextPrefix = strings.Repeat("\t", depth+1)
	}

	switch v.kind {
	case StringKind:
		return strconv.Quote(v.v.(string))

	case NullKind:
		return "null"

	case NumberKind:
		return v.v.(*big.Float).Text('g', 6)

	case FuncKind:
		return `"<func>"`

	case ObjectKind:
		m := v.v.(Mapper)
		if m.Len() == 0 {
			return "{}"
		}

		var sb strings.Builder
		sb.WriteString("{")

		idx := 0
		for key, val := range m.All() {
			if idx != 0 {
				sb.WriteByte(',')
			}
			idx++

			if indent {
				sb.WriteByte('\n')
			}
			sb.WriteString(nextPrefix)
			sb.WriteString(strconv.Quote(key.String()))
			sb.WriteString(": ")
			sb.WriteString(fn(val, depth+1, indent))
		}

		if indent {
			sb.WriteByte('\n')
			sb.WriteString(prefix)
		}
		sb.WriteByte('}')

		return sb.String()

	case ArrayKind:
		var vals []Value
		for val := range v.v.(Indexer).Values() {
			vals = append(vals, val)
		}

		if len(vals) == 0 {
			return "[]"
		}

		// check if all elements are simple
		allSimple := true
		for _, val := range vals {
			if val.kind == ObjectKind || val.kind == ArrayKind {
				allSimple = false
				break
			}
		}

		var sb strings.Builder
		sb.WriteByte('[')

		if allSimple && len(vals) <= 5 {
			// inline small arrays of simple values
			for i, val := range vals {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fn(val, depth+1, indent))
			}
		} else {
			// multi-line for complex or large arrays
			for i, val := range vals {
				sb.WriteByte('\n')
				sb.WriteString(nextPrefix)
				sb.WriteString(fn(val, depth+1, indent))
				if i < len(vals)-1 {
					sb.WriteByte(',')
				}
			}
			sb.WriteByte('\n')
			sb.WriteString(prefix)
		}
		sb.WriteByte(']')

		return sb.String()

	default:
		return fmt.Sprintf("%v", v.v)
	}
}
