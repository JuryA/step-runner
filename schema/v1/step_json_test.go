package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestStepSchemaValidate(t *testing.T) {
	cases := []struct {
		name    string
		step    string
		wantErr bool
	}{{
		name: "local step",
		step: `
name: my_step
step: ./my-step
`,
	}, {
		name: "remote step",
		step: `
name: my_step
step: gitlab.com/my-org/my-step@v1
`,
	}, {
		name: "remote nested step",
		step: `
name: my_step
step:
  git:
    url: gitlab.com/my-org/my-step
    rev: v1
`,
	}, {
		name:    "remote nested step missing rev",
		wantErr: true,
		step: `
name: my_step
step:
  git:
    url: gitlab.com/my-org/my-step
`,
	}, {
		name:    "remote nested step missing url",
		wantErr: true,
		step: `
name: my_step
step:
  git:
    rev: v1
`,
	}, {
		name: "remote nested step with dir",
		step: `
name: my_step
step:
  git:
    url: gitlab.com/my-org/my-step
    rev: v1
    dir: sub-dir
    file: my-step.yml
`,
	}, {
		name:    "remote nested step with additional properties",
		wantErr: true,
		step: `
name: my_step
step:
  git:
    additional: property
`,
	}, {
		name: "script step",
		step: `
name: my_step
script: my-script
`,
	}, {
		name:    "empty script step",
		wantErr: true,
		step: `
name: my_step
script: ""
`,
	}, {
		name: "remote action",
		step: `
name: my_step
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
		name:    "exec with empty command",
		wantErr: true,
		step: `
exec:
  command: []
`,
	}, {
		name: "exec with work dir",
		step: `
exec: 
  command: [ my-binary ]
  work_dir: sub-dir
`,
	}, {
		name:    "empty step invalid",
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
		name:    "step mutually exclusive with run",
		wantErr: true,
		step: `
step: ./my-step
run:
  - step: ./my-step
`,
	}, {
		name:    "script mutually exclusive with action",
		wantErr: true,
		step: `
script: echo hello world
action: my-action@v1
`,
	}, {
		name:    "script mutually exclusive with exec",
		wantErr: true,
		step: `
script: echo hello world
exec:
  command: [ my-binary ]
`,
	}, {
		name:    "script mutually exclusive with run",
		wantErr: true,
		step: `
script: echo hello world
run:
  - step: ./my-step
`,
	}, {
		name:    "action mutually exclusive with exec",
		wantErr: true,
		step: `
action: my-action@v1
exec:
  command: [ my-binary ]
`,
	}, {
		name:    "action mutually exclusive with run",
		wantErr: true,
		step: `
action: my-action@v1
run:
  - step: ./my-step
`,
	}, {
		name:    "exec mutually exclusive with run",
		wantErr: true,
		step: `
exec:
  command: [ my-binary ]
run:
  - step: ./my-step
`,
	}, {
		name:    "mutual exclusion recursively",
		wantErr: true,
		step: `

run:
  - step: gitlab.com/components/my-step
    exec:
      command: [echo, "hello world"]
    run:
      - step: gitlab.com/components/another-step
`,
	}, {
		name: "name must be alphanumeric",
		step: `
run:
    - name: not allowed to have a space
      script: echo hello world
`,
		wantErr: true,
	}, {
		name: "env names must be alphanumeric",
		step: `
run:
    - name: my_step
      step: ./my-step
      env:
          invalid name: foo
`,
		wantErr: true,
	}, {
		name: "output names must be alphanumeric",
		step: `
run:
    - name: my_step
      step: ./my-step
outputs:
    invalid name: foo
`,
		wantErr: true,
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
