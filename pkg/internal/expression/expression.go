package expression

import (
	"fmt"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/structpb"
	"strings"
)

func Evaluate(obj any, s string) (*context.Variable, error) {
	s = strings.TrimSpace(s)
	fields := strings.Split(s, ".")

	value, err := evaluate(obj, s)

	if err != nil {
		return nil, err
	}

	isSensitive, err := fieldIsSensitive(obj, fields)

	if err != nil {
		return nil, err
	}

	return context.NewVariable(value, isSensitive), nil
}

func evaluate(obj any, s string) (*structpb.Value, error) {
	for _, key := range strings.Split(s, ".") {
		res, err := DigObject(obj, key)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", s, err)
		}
		obj = res
	}

	return ObjectToProtoValue(obj)
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
