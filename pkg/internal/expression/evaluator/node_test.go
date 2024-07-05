package evaluator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/evaluator"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

type testContext struct {
	Value string `json:"v"`
}

func TestCompileStatement(t *testing.T) {
	context := &testContext{
		Value: "test",
	}

	tests := []struct {
		name   string
		text   string
		result value.Value
	}{
		{name: "and expression", text: "1 && 2", result: value.ToValue(true)},
		{name: "or expression", text: "1 || 2", result: value.ToValue(true)},
		{name: "not equals expression", text: "1 != 2", result: value.ToValue(true)},
		{name: "equals expression", text: "1 == 2", result: value.ToValue(false)},
		{name: "equals expression", text: "\"1\" == 1", result: value.ToValue(true)},
		{name: "equals expression", text: "\"1\" == \"1\"", result: value.ToValue(true)},
		{name: "equals expression", text: "\"1\" == \"2\"", result: value.ToValue(false)},
		{name: "and expression", text: "0 && 2", result: value.ToValue(false)},
		{name: "or expression", text: "0 || 2", result: value.ToValue(true)},
		{name: "and expression", text: "\"\" && 2", result: value.ToValue(false)},
		{name: "or expression", text: "\"\" || 2", result: value.ToValue(true)},
		{name: "variable", text: "v", result: value.ToValue("test")},
		{name: "str(value)", text: "str(10)", result: value.ToValue("10")},
		{name: "self.str()", text: "10.str()", result: value.ToValue("10")},
		{name: "self.orDefault()", text: "10.orDefault(20)", result: value.ToValue(10)},
		{name: "self.orDefault()", text: "0.orDefault(20)", result: value.ToValue(20)},
		{name: "self.orDefault()", text: "non_existing.orDefault(20)", result: value.ToValue(20)},
		{name: "self.orDefault()", text: "non_existing.orDefault(20) && v", result: value.ToValue(true)},
		{name: "conditional", text: "1 ? 2 : 3", result: value.ToValue(2)},
		{name: "conditional", text: "0 ? 2 : 3", result: value.ToValue(3)},
		{name: "conditional", text: "v ? 2 : 3", result: value.ToValue(2)},
		{name: "conditional", text: "non_existing ? 2 : 3", result: value.ToValue(3)},
		{name: "coalesce", text: "0 ?: 0 ?: 0 ?: 2", result: value.ToValue(2)},
		{name: "coalesce", text: "1 ?: 2", result: value.ToValue(1)},
		{name: "coalesce", text: "0 ?: 2", result: value.ToValue(2)},
		{name: "coalesce", text: "v ?: 2", result: value.ToValue("test")},
		{name: "coalesce", text: "non_existing ?: 2", result: value.ToValue(2)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node, err := evaluator.CompileStatement(test.text)
			if err != nil {
				assert.NoError(t, err)
				return
			}

			result := node.Calculate(value.ToValue(context))
			resultStr, _ := result.ToString()
			assert.EqualValues(t, test.result, result, "result should be equal: %s", resultStr)
		})
	}
}
