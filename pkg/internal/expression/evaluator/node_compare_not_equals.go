package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeCompareNotEquals struct {
	left, right Node
}

func (n *nodeCompareNotEquals) Calculate(context value.Value) value.Value {
	left := n.left.Calculate(context)
	right := n.right.Calculate(context)
	result := value.Equals(left, right)
	if result.Error() != nil {
		return result
	}
	return value.ToValue(!result.IsTrue())
}
