package evaluator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/evaluator"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

func TestCompileStatement(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		result value.Value
	}{
		{name: "and expression", text: "1 && 2", result: value.ToValue(2)},
		{name: "or expression", text: "1 || 2", result: value.ToValue(1)},
		{name: "equals expression", text: "1 == 2", result: value.ToValue(false)},
		{name: "not equals expression", text: "1 != 2", result: value.ToValue(true)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node, err := evaluator.CompileStatement(test.text)
			if err != nil {
				assert.NoError(t, err)
				return
			}

			result := node.Calculate(nil)
			assert.EqualValues(t, test.result, result)
		})
	}
}
