package e2e_tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestWriteToStruct(t *testing.T) {
	t.Run("struct output", func(t *testing.T) {
		execStep := `
spec:
  outputs:
    out_value:
      type: struct
---
exec:
  command:
    - sh
    - -c
    - 'echo "{\"name\":\"out_value\",\"value\":{\"key1\":\"value1\"} }" >>${{output_file}}'`
		stepDir := bldr.Files(t).WriteFile("step.yml", execStep).Build()

		runStep := `
spec:
---
run:
  - name: exec_step
    step: %s
  - name: echo_output
    script: 'echo "Output of key1 is ${{steps.exec_step.outputs.out_value.key1}}"'`
		_, log, err := testutil.StepRunner(t).Run(fmt.Sprintf(runStep, stepDir))
		require.NoError(t, err)
		require.Contains(t, log, "Output of key1 is value1")
	})

	t.Run("struct as an input", func(t *testing.T) {
		// fails because struct is turns into {"name":"out_value","value":{foo:bar}}
		execStep := `
spec:
  inputs:
    in_value:
      type: struct
  outputs:
    out_value:
      type: struct
---
exec:
  command:
    - sh
    - -c
    - 'echo "{\"name\":\"out_value\",\"value\":${{inputs.in_value}}}" >>${{output_file}}'`
		stepDir := bldr.Files(t).WriteFile("step.yml", execStep).Build()

		runStep := `
spec:
---
run:
  - name: exec_step
    step: %s
    inputs:
      in_value:
        foo: bar
  - name: echo_output
    script: 'echo "foo is ${{steps.exec_step.outputs.out_value.foo}}"'`

		_, log, err := testutil.StepRunner(t).Run(fmt.Sprintf(runStep, stepDir))
		require.NoError(t, err)
		require.Contains(t, log, "foo is bar")
	})

	t.Run("struct as an input, each input is individually specified", func(t *testing.T) {
		execStep := `
spec:
  inputs:
    in_value:
      type: struct
  outputs:
    out_value:
      type: struct
---
exec:
  command:
    - sh
    - -c
    - 'echo "{\"name\":\"out_value\",\"value\":{\"foo\":\"${{inputs.in_value.foo}}\"} }" >>${{output_file}}'`
		stepDir := bldr.Files(t).WriteFile("step.yml", execStep).Build()

		runStep := `
spec:
---
run:
  - name: exec_step
    step: %s
    inputs:
      in_value:
        foo: bar
  - name: echo_output
    script: 'echo "foo is ${{steps.exec_step.outputs.out_value.foo}}"'`

		_, log, err := testutil.StepRunner(t).Run(fmt.Sprintf(runStep, stepDir))
		require.NoError(t, err)
		require.Contains(t, log, "foo is bar")
	})
}
