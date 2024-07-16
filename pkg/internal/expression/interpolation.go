package expression

import (
	"fmt"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"regexp"

	"google.golang.org/protobuf/types/known/structpb"
)

const InterpolateOpen = "${{"
const InterpolateClose = "}}"

var interpolateRegex = regexp.MustCompile(regexp.QuoteMeta(InterpolateOpen) + "|" + regexp.QuoteMeta(InterpolateClose))

func interpolateString(obj any, value string) (*context.Variable, error) {
	output := []*structpb.Value{}
	depth := 0
	prev_idx := 0
	open_idx := 0

	for _, loc := range interpolateRegex.FindAllStringIndex(value, -1) {
		if depth == 0 && prev_idx != loc[0] {
			// add prefix to output
			output = append(output, structpb.NewStringValue(value[prev_idx:loc[0]]))
		}
		prev_idx = loc[1]

		switch value[loc[0]:loc[1]] {
		case InterpolateOpen:
			depth += 1

			if depth == 1 {
				open_idx = loc[1]
			}

		case InterpolateClose:
			depth -= 1

			insideString := value[open_idx:loc[0]]

			if depth < 0 {
				return nil, fmt.Errorf("The %q has extra '}}'", insideString)
			} else if depth > 0 {
				break
			}

			insideValue, err := Evaluate(obj, insideString)
			if err != nil {
				return nil, err
			}
			output = append(output, insideValue.Value)

		default:
		}
	}

	if depth > 0 {
		return nil, fmt.Errorf("The %q is not closed: ${{ ... }}", value[open_idx:])
	}

	// add suffix to output
	if prev_idx != len(value) {
		output = append(output, structpb.NewStringValue(value[prev_idx:]))
	}

	// retain type if this is single item, otherwise convert to string
	if len(output) == 1 {
		return context.NewVariable(output[0], false), nil
	}

	// concat all items
	res := ""
	for _, o := range output {
		str, err := ValueToString(o)
		if err != nil {
			return nil, err
		}
		res += str
	}

	return context.NewStringVariable(res, false), nil
}

func expandStruct(obj any, value *structpb.Struct) (*context.Variable, error) {
	res := &structpb.Struct{Fields: make(map[string]*structpb.Value, len(value.Fields))}

	for fieldKey, fieldValue := range value.Fields {
		fieldNewValue, err := Expand(obj, fieldValue)
		if err != nil {
			return nil, err
		}
		res.Fields[fieldKey] = fieldNewValue.Value
	}
	return context.NewStructVariable(res, false), nil
}

func expandList(obj any, value *structpb.ListValue) (*context.Variable, error) {
	res := &structpb.ListValue{Values: make([]*structpb.Value, len(value.Values))}

	for listIndex, listValue := range value.Values {
		listNewValue, err := Expand(obj, listValue)
		if err != nil {
			return nil, err
		}
		res.Values[listIndex] = listNewValue.Value
	}
	return context.NewListVariable(res, false), nil
}

// The Expand rewrites struct/list/string mutating data structure
func Expand(obj any, value *structpb.Value) (*context.Variable, error) {
	switch value.Kind.(type) {
	case *structpb.Value_StringValue:
		return interpolateString(obj, value.GetStringValue())

	case *structpb.Value_StructValue:
		return expandStruct(obj, value.GetStructValue())

	case *structpb.Value_ListValue:
		return expandList(obj, value.GetListValue())

	default:
		return context.NewVariable(value, false), nil
	}
}

// The ExpandString rewrites string and returns string
func ExpandString(obj any, value string) (string, error) {
	res, err := interpolateString(obj, value)
	if err != nil {
		return "", err
	}

	return ValueToString(res.Value)
}
