package context

import (
	"google.golang.org/protobuf/types/known/structpb"
	"strings"
)

type Variable struct {
	Value     *structpb.Value
	Sensitive bool
}

func NewVariable(value *structpb.Value, sensitive bool) *Variable {
	strings.NewReplacer()

	if value == nil {
		panic("variable must have a value")
	}

	variable := &Variable{
		Value:     value,
		Sensitive: sensitive,
	}

	return variable
}

func NewStringVariable(value string, sensitive bool) *Variable {
	return NewVariable(structpb.NewStringValue(value), sensitive)
}

func NewStructVariable(value *structpb.Struct, sensitive bool) *Variable {
	return NewVariable(structpb.NewStructValue(value), sensitive)
}

func NewListVariable(value *structpb.ListValue, sensitive bool) *Variable {
	return NewVariable(structpb.NewListValue(value), sensitive)
}
