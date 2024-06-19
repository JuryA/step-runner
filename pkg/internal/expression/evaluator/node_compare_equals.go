package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeCompareEquals struct {
	left, right Node
}

func (n *nodeCompareEquals) Calculate(context value.Value) value.Value {
	left := n.left.Calculate(context)
	right := n.right.Calculate(context)
	return value.Equals(left, right)
}
