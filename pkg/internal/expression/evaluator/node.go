package evaluator

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

type Node interface {
	Calculate(context value.Value) value.Value
}

func CompileStatement(text string) (Node, error) {
	lexer := newExprLexer([]byte(text))
	status := exprParse(lexer)
	if status != 0 {
		return nil, fmt.Errorf("Parse failure: %d", status)
	}

	return lexer.result, nil
}
