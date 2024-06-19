package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeValue struct {
	value value.Value
}

func (n *nodeValue) Calculate(context value.Value) value.Value {
	return n.value
}
