package evaluator

import (
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

type nodeCall struct {
	expr   Node
	method string
	args   []Node
}

func (n *nodeCall) Calculate(context value.Value) value.Value {
	exprValue := n.expr.Calculate(context)
	if exprValue.Error() != nil {
		return exprValue
	}

	args := []value.Value{}
	for _, arg := range n.args {
		argValue := arg.Calculate(context)
		if argValue.Error() != nil {
			return argValue
		}
		args = append(args, argValue)
	}

	return exprValue.Call(n.method, args)
}
