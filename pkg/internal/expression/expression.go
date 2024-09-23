package expression

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
)

func Evaluate(obj any, s string) (*context.Value, error) {
	s = strings.TrimSpace(s)
	value, err := evaluate(obj, s)

	if err != nil {
		return nil, err
	}

	return context.NewValue(value, false, s), nil
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
