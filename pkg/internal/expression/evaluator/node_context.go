package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeContext struct {
}

func (n *nodeContext) Calculate(context value.Value) value.Value {
	return context
}
