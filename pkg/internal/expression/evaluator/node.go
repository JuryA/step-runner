package evaluator

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

type Node interface {
	Calculate(context value.Value) value.Value
}

func CompileStatement(text string) (Node, error) {
	parser := newExpressionParser([]byte(text))
	status := exprParse(parser)
	if len(parser.errors) > 0 {
		return nil, fmt.Errorf("Parse errors: %v", parser.errors)
	}
	if status != 0 {
		return nil, fmt.Errorf("Parse failure: %d", status)
	}

	return parser.result, nil
}
