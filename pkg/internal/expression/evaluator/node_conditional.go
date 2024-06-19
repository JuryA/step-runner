package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeConditional struct {
	check, left, right Node
}

func (n *nodeConditional) Calculate(context value.Value) value.Value {
	check := n.check.Calculate(context)
	if check.IsTrue() {
		return n.left.Calculate(context)
	} else {
		return n.right.Calculate(context)
	}
}
