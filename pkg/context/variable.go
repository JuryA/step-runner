package context

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

// Variable represents an expression that is dynamically evaluated. Variables are immutable.
// e.g. 'greeting' in the following is a Variable:
//   - step: ./say-hello
//     inputs:
//     greeting: Hello, ${{ env.USERNAME }}
type Variable struct {
	Value     *structpb.Value // Value is an expression or the result of an expression
	Sensitive bool            // Whether the variable holds sensitive data
}

func NewVariable(value *structpb.Value, sensitive bool) *Variable {
	if value == nil {
		panic("variable must have a value")
	}

	return &Variable{
		Value:     value,
		Sensitive: sensitive,
	}
}

func NewStringVariable(value string, sensitive bool) *Variable {
	return NewVariable(structpb.NewStringValue(value), sensitive)
}

func (v *Variable) Assign(value *Value) (*Variable, error) {
	if value.Sensitive && !v.Sensitive {
		return nil, fmt.Errorf("non-sensitive input cannot derive value using sensitive value(s) %q", value.SensitiveReason)
	}

	return NewVariable(value.Value, value.Sensitive), nil
}
