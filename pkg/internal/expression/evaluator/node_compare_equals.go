package evaluator

import "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"

type nodeCompareEquals struct {
	left, right Node
}

func (n *nodeCompareEquals) Calculate(context value.Value) value.Value {
	left := n.left.Calculate(context)
	right := n.right.Calculate(context)

	value1String, value1Err := left.ToString()
	value2String, value2Err := right.ToString()
	if value1Err != nil && value2Err != nil {
		return value.NewError("Many errors: %q, %q", value1Err, value2Err)
	} else if value1Err != nil {
		return value.NewError(value1Err)
	} else if value2Err != nil {
		return value.NewError(value2Err)
	} else {
		return value.ToValue(value1String == value2String)
	}
}
