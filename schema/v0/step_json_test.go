package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestStepSchemaValidate(t *testing.T) {

	// step
	// script
	// action
	// exec
	// steps

	cases := []struct {
		name    string
		step    string
		wantErr bool
	}{{
		name: "local step",
		step: `
step: ./my-step
`,
	}, {
		name: "script step",
		step: `
script: my-script
`,
	}, {
		name: "remote action",
		step: `
action: my-action@v1
`,
	}, {
		name: "exec",
		step: `
exec:
  command: [ my-binary ]
`,
	}, {
		name:    "exec without command",
		wantErr: true,
		step: `
exec: {}
`,
	}, {
		name:    "empty invalid",
		wantErr: true,
		step:    "",
	}, {
		name:    "step mutually exclusive with script",
		wantErr: true,
		step: `
script: echo hello world
action: my-action@v1
`,
	}, {
		name:    "step mutually exclusive with action",
		wantErr: true,
		step: `
step: ./my-step
action: my-action@v1
`,
	}, {
		name:    "step mutually exclusive with exec",
		wantErr: true,
		step: `
step: ./my-step
exec:
  command: [ bash, -c, "echo hello world" ]
`,
	}, {
		name:    "step mutually exclusive with steps",
		wantErr: true,
		step: `
step: ./my-step
steps:
  - name: my_step
    step: ./my-step
`,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var untyped any
			err := yaml.Unmarshal([]byte(c.step), &untyped)
			require.NoError(t, err)
			if c.wantErr {
				require.Error(t, stepSchema.Validate(untyped))
			} else {
				require.NoError(t, stepSchema.Validate(untyped))
			}
		})
	}

}
