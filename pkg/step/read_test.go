package step

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/schema"
)

func TestRead(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		wantSpec *schema.Spec
		wantDef  *schema.Definition
		wantErr  bool
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
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stepDef, err := ReadSteps(c.yaml, "")
			if c.wantErr {
				require.Error(t, err)
				require.Nil(t, stepDef)
			} else {
				require.NoError(t, err)
				if !reflect.DeepEqual(c.wantSpec, stepDef.Spec) {
					t.Errorf("wanted:\n%+v\ngot:\n%+v", c.wantSpec, stepDef.Spec)
				}
				if !reflect.DeepEqual(c.wantDef, stepDef.Definition) {
					t.Errorf("wanted:\n%+v\ngot:\n%+v", c.wantDef, stepDef.Definition)
				}
			}
		})
	}
}
