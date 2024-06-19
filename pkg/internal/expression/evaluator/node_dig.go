package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeDig struct {
	expr Node
	key  string
}

func (n *nodeDig) Calculate(context value.Value) value.Value {
	return n.expr.Calculate(context).Dig(n.key)
}
