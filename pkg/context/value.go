package context

import (
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// Value represents a result of an expression.
// e.g. the result of 'Hello, ${{ env.USERNAME }' when all values have been interpolated.
type Value struct {
	Value           *structpb.Value
	Sensitive       bool
	SensitiveReason string
}

func NewValue(value *structpb.Value, sensitive bool, sensitiveReasons ...string) *Value {
	if value == nil {
		panic("value must have an inner value")
	}

	return &Value{
		Value:           value,
		Sensitive:       sensitive,
		SensitiveReason: strings.Join(sensitiveReasons, ","),
	}
}

func NewStringValue(value string, sensitive bool, sensitiveReasons ...string) *Value {
	return NewValue(structpb.NewStringValue(value), sensitive, sensitiveReasons...)
}

func NewNonSensitiveValue(value *structpb.Value) *Value {
	return NewValue(value, false)
}
