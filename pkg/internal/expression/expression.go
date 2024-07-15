package expression

import (
	"fmt"
	"slices"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

func Evaluate(obj any, s string) (*structpb.Value, bool, error) {
	s = strings.TrimSpace(s)
	fields := strings.Split(s, ".")

	isSensitive, err := fieldIsSensitive(obj, fields)

	if err != nil {
		return nil, false, fmt.Errorf("failed to evaluate %s due to %w", s, err)
	}

	value, err := evaluate(obj, s)
	return value, isSensitive, err
}

func evaluate(obj any, s string) (*structpb.Value, error) {
	for _, key := range strings.Split(s, ".") {
		res, err := DigObject(obj, key)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", s, err)
		}
		obj = res
	}

	value, err := ObjectToProtoValue(obj)
	return value, err
}

func fieldIsSensitive(obj any, fields []string) (bool, error) {
	outputIndex := slices.Index(fields, "outputs")

	if outputIndex < 0 || outputIndex > len(fields)-1 {
		return false, nil
	}

	pathToOutputSpec := fields[0:outputIndex]
	pathToOutputSpec = append(pathToOutputSpec, "specDefinition", "spec", "spec", "outputs", fields[outputIndex+1], "sensitive")
	value, err := evaluate(obj, strings.Join(pathToOutputSpec, "."))

	if err != nil {
		return false, fmt.Errorf("failed to determine if field is sensitive: %w", err)
	}

	if value == nil {
		return false, nil
	}

	return value.GetBoolValue(), nil
}
