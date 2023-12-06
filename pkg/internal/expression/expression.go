package expression

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

func Evaluate(obj any, s string) (*structpb.Value, error) {
	s = strings.TrimSpace(s)
	for _, key := range strings.Split(s, ".") {
		res, err := DigObject(obj, key)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", s, err)
		}
		obj = res
	}

	return ObjectToProtoValue(obj)
}
