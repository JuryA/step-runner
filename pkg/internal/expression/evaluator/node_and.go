package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeAnd struct {
	left, right Node
}

func (n *nodeAnd) Calculate(context value.Value) value.Value {
	left := n.left.Calculate(context)
	if left.Error() != nil {
		return left
	} else if !left.IsTrue() {
		return value.ToValue(nil)
	}

	return n.right.Calculate(context)
}
