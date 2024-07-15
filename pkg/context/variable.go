package context

import "google.golang.org/protobuf/types/known/structpb"

type Variable struct {
	Value     *structpb.Value
	Sensitive bool
}

func NewVariable(value *structpb.Value, sensitive bool) *Variable {
	variable := &Variable{
		Value:     value,
		Sensitive: sensitive,
	}

	if value == nil {
		panic("variable must have a value")
	}

	return variable
}

func NewStringVariable(value string, sensitive bool) *Variable {
	return &Variable{
		Value:     structpb.NewStringValue(value),
		Sensitive: sensitive,
	}
}

func NewStructVariable(value *structpb.Struct, sensitive bool) *Variable {
	return &Variable{
		Value:     structpb.NewStructValue(value),
		Sensitive: sensitive,
	}
}

func NewListVariable(value *structpb.ListValue, sensitive bool) *Variable {
	return &Variable{
		Value:     structpb.NewListValue(value),
		Sensitive: sensitive,
	}
}
