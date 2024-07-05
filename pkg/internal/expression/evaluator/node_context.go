package evaluator

import (
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

type valueContext struct {
	context value.Value
}

func (v *valueContext) Dig(key string) value.Value {
	switch key {
	case "self":
		return v.context
	}
	return v.context.Dig(key)
}

func (v *valueContext) Call_str(_self value.Value, arg value.Value) value.Value {
	x, err := arg.ToString()
	if err != nil {
		return value.NewError(err)
	}
	return value.ToValue(x)
}

func (v *valueContext) IsTrue() bool {
	return v.context.IsTrue()
}

func (v *valueContext) IsNull() bool {
	return v.context.IsNull()
}

func (v *valueContext) Error() error {
	return v.context.Error()
}

func (v *valueContext) ToString() (string, error) {
	return v.context.ToString()
}

type nodeContext struct {
}

func (n *nodeContext) Calculate(context value.Value) value.Value {
	if context == nil {
		context = &value.ValueNil{}
	}
	return &valueContext{context: context}
}
