package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeUnaryNot struct {
	expr Node
}

func (n *nodeUnaryNot) Calculate(context value.Value) value.Value {
	expr := n.expr.Calculate(context)
	if expr.Error() != nil {
		return expr
	}
	return value.ToValue(!expr.IsTrue())
}
