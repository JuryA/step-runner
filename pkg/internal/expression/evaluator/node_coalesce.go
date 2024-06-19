package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeCoalesce struct {
	left, right Node
}

func (n *nodeCoalesce) Calculate(context value.Value) value.Value {
	left := n.left.Calculate(context)
	if left.IsTrue() {
		return left
	} else {
		return n.right.Calculate(context)
	}
}
