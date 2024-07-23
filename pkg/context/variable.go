package context

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

// Variable represents an expression that is dynamically evaluated. Variables are mutable.
// e.g. 'greeting' in the following is a Variable:
//   - step: ./say-hello
//     inputs:
//     greeting: Hello, ${{ env.USERNAME }}
type Variable struct {
	Value     *structpb.Value // Value is initially an expression, and can be updated to be the result of the expression.
	Sensitive bool            // Whether the variable can hold sensitive data
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

func (v *Variable) Assign(value *Value) error {
	if value.Sensitive && !v.Sensitive {
		return fmt.Errorf("non-sensitive input cannot derive value using sensitive value(s) %q", value.SensitiveReason)
	}

	v.Value = value.Value
	return nil
}
