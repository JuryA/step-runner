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
