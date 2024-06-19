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
		return value.ToValue(false)
	}

	right := n.right.Calculate(context)
	if right.Error() != nil {
		return right
	} else if !right.IsTrue() {
		return value.ToValue(false)
	}

	return value.ToValue(true)
}
