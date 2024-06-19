package evaluator

import (
	"errors"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

type nodeCall struct {
	expr   Node
	method string
	args   []Node
}

func (n *nodeCall) Calculate(context value.Value) value.Value {
	return value.NewError(errors.New("not supported"))
}
