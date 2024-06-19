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
		{name: "not equals expression", text: "1 != 2", result: value.ToValue(true)},
		{name: "equals expression", text: "1 == 2", result: value.ToValue(false)},
		{name: "equals expression", text: "\"1\" == 1", result: value.ToValue(true)},
		{name: "equals expression", text: "\"1\" == \"1\"", result: value.ToValue(true)},
		{name: "equals expression", text: "\"1\" == \"2\"", result: value.ToValue(false)},
		{name: "and expression", text: "0 && 2", result: value.ToValue(nil)},
		{name: "or expression", text: "0 || 2", result: value.ToValue(2)},
		{name: "and expression", text: "\"\" && 2", result: value.ToValue(nil)},
		{name: "or expression", text: "\"\" || 2", result: value.ToValue(2)},
		{name: "variable", text: "v", result: value.ToValue(nil)},
		{name: "str()", text: "str(10)", result: value.ToValue("10")},
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
