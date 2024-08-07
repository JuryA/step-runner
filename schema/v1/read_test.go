package schema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
steps:
    - name: ""
      script: echo hello world
`,
	}, {
		name: "everything step",
		yaml: `
spec:
    inputs:
        age:
            type: number
            default: 12
        favorites:
            type: struct
            default:
                food: apple
        name:
            type: string
            default: foo
---
exec:
    command:
        - echo
        - hello world
`,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stepDef, err := ReadSteps(c.yaml, "")
			if c.wantErr {
				require.Error(t, err)
				require.Nil(t, stepDef)
			} else {
				require.NoError(t, err)
				// Assert that the whole step is preserved round-trip
				got, err := WriteSteps(stepDef)
				require.NoError(t, err)
				want := strings.TrimSpace(c.yaml)
				got = strings.TrimSpace(got)
				require.Equal(t, want, got)
			}
		})
	}
}
