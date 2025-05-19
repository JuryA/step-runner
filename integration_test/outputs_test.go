package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
)

func TestStepOutputs(t *testing.T) {
	t.Run("can access outputs of a previous step", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: greet_foo
    step: ./steps/greeting
    inputs:
      name: foo
  - name: greet_previous
    step: ./steps/greeting
    inputs:
      name: ${{steps.greet_foo.outputs.name}}
`
		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 2)
		require.Equal(t, "foo", result.SubStepResults[0].Outputs["name"].GetStringValue())
		require.Equal(t, "foo", result.SubStepResults[1].Outputs["name"].GetStringValue())
	})

	t.Run("can access outputs of a composite step", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: greet_the_crew
    step: ./steps/crew
    inputs: {}
  - name: greet_previous
    step: ./steps/greeting
    inputs:
      name: ${{steps.greet_the_crew.outputs.crew_name_1}}
`
		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 2)
		require.Equal(t, "sponge bob", result.SubStepResults[0].Outputs["crew_name_1"].GetStringValue())
		require.Equal(t, "sponge bob", result.SubStepResults[1].Outputs["name"].GetStringValue())
	})

	t.Run("cannot access outputs of composite children", func(t *testing.T) {
		yaml := `
spec: {}
---
run:
  - name: greet_the_crew
    step: ./steps/crew
    inputs: {}
  - name: greet_previous
    step: ./steps/greeting
    inputs:
      name: ${{steps.greet_sponge_bob.outputs.name}}`

		_, _, err := testutil.StepRunner(t).Run(yaml)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "greet_previous": failed to load: expand input "name": steps.greet_sponge_bob.outputs.name: the "greet_sponge_bob" was not found`)
	})

	t.Run("sequence of steps returns outputs", func(t *testing.T) {
		yaml := `
spec:
  outputs:
    name:
      type: string
---
run:
  - name: do_nothing
    step: ./steps/exit
outputs:
  name: "Foo"`

		result, _, err := testutil.StepRunner(t).Run(yaml)
		require.NoError(t, err)
		require.Equal(t, "Foo", result.Outputs["name"].GetStringValue())
	})
}
