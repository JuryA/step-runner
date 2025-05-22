package schema_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

func TestRead(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr bool
	}{{
		name: "documents out of order",
		yaml: `
type: exec
exec:
  command: [echo, "${{inputs.name}}"]
---
spec:
  inputs:
    name:
`,
		wantErr: true,
	}, {
		name: "missing spec",
		yaml: `
type: exec
exec:
  command: [echo, "${{inputs.name}}"]
`,
		wantErr: true,
	}, {
		name: "missing definition",
		yaml: `
spec:
  inputs:
    name:
`,
		wantErr: true,
	}, {
		name: "minimal step",
		yaml: `
{}
---
run:
    - name: my_step
      script: echo hello world
`,
	}, {
		name: "everything step",
		yaml: `
spec:
    inputs:
        age:
            default: 12
            type: number
        favorites:
            default:
                food: apple
            type: struct
        name:
            default: foo
            type: string
---
exec:
    command:
        - echo
        - hello world
`,
	}, {
		name: "step with description",
		yaml: `
description: This is a test step
spec:
    inputs:
        name:
            description: This is a test input
            type: string
---
exec:
    command:
        - echo
        - hello world
`,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec, step, err := schema.ReadSteps(c.yaml)
			if c.wantErr {
				require.Error(t, err)
				require.Nil(t, spec)
				require.Nil(t, step)
			} else {
				require.NoError(t, err)
				// Assert that the whole step is preserved round-trip
				got, err := schema.WriteSteps(spec, step)
				require.NoError(t, err)
				want := strings.TrimSpace(c.yaml)
				got = strings.TrimSpace(got)
				require.Equal(t, want, got)
			}
		})
	}
}
