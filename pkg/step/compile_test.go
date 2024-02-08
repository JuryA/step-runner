package step

import (
	"testing"

	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name         string
		steps        string
		wantCompiled string
		wantErr      bool
	}{{
		name: "simple case",
		steps: `
spec:
    inputs:
        name:
---
type: exec
exec:
    command:
        - echo
        - ${{inputs.name}}
`,
		wantCompiled: `
spec:
    inputs:
        name:
            type: string
---
type: exec
exec:
    command:
        - echo
        - ${{inputs.name}}
`,
	}, {
		name: "top level script compiles to a single step",
		steps: `
spec:
---
script: echo hello world
`,
		wantCompiled: `
spec:
    inputs: {}
---
type: steps
steps:
    - name: "run a script"
      step: "https://gitlab.com/josephburnett/script@v1" # until we create the canonical step repository
      inputs:
          script: echo hello world
`,
	}, {
		name: "cannot set definition type with top level script",
		steps: `
spec:
---
script: echo hello world
type: steps
`,
		wantErr: true,
	}, {
		name: "cannot set definition steps with top level script",
		steps: `
spec:
---
script: echo hello world
steps: []
`,
		wantErr: true,
	}, {
		name: "step script keyword compiles to a single step",
		steps: `
spec:
---
type: steps
steps:
  - name: "my special script name"
    script: echo hello world
`,
		wantCompiled: `
spec:
    inputs: {}
---
type: steps
steps:
    - name: "my special script name"
      step: "https://gitlab.com/josephburnett/script@v1" # until we create the canonical steps repository
      inputs:
          script: echo hello world
`,
	}, {
		name: "complex type: exec",
		steps: `
spec:
    inputs:
        age:
            default: 1
            type: number
        favorites:
            default:
                color: red
            type: struct
        hungry:
            default: false
            type: bool
        name:
            default: steppy
            type: string
    outputs:
        eye_color:
            default: brown
---
exec:
    command:
        - echo
        - meet ${{inputs.name}}
        - who is ${{inputs.age}}
        - likes ${{inputs.favorites}}
        - and is hungry (${{inputs.hungry}}).
type: exec
`,
		wantCompiled: `
spec:
    inputs:
        age:
            default: 1
            type: number
        favorites:
            default:
                color: red
            type: struct
        hungry:
            default: false
            type: bool
        name:
            default: steppy
            type: string
    outputs:
        eye_color:
            default: brown
---
exec:
    command:
        - echo
        - meet ${{inputs.name}}
        - who is ${{inputs.age}}
        - likes ${{inputs.favorites}}
        - and is hungry (${{inputs.hungry}}).
type: exec
`,
	}, {
		name: "complex type: steps",
		steps: `
spec: {}
---
outputs:
    eye_color: brown
steps:
    - env:
        JOB_ID: ${{job.id}}
        USER: srunner
      inputs:
        age: 1
        favorites:
            food:
                - hamburger
                - sausage
        hungry: false
        name: steppy
      name: foo_to_the_max
      step: git+https://gitlab.com/gitlab-org/foo@v1
    - inputs:
        greeting: ${{steps.foo to the max.outputs.greeting}}
      name: foo_redux
      step: ../steps/redux
type: steps
`,
		wantCompiled: `
spec: {}
---
type: steps
steps:
    - name: foo_to_the_max
      step: git+https://gitlab.com/gitlab-org/foo@v1
      env:
        JOB_ID: ${{job.id}}
        USER: srunner
      inputs:
        age: 1
        favorites:
            food:
                - hamburger
                - sausage
        hungry: false
        name: steppy
    - name: foo_redux
      step: ../steps/redux
      inputs:
        greeting: ${{steps.foo to the max.outputs.greeting}}
outputs:
    eye_color: brown
`,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stepDef, err := ReadSteps(c.steps, "")
			require.NoError(t, err)
			protoStepDef, err := CompileSteps(stepDef)
			if c.wantErr {
				require.Error(t, err)
				require.Nil(t, protoStepDef)
			} else {
				require.NoError(t, err)
				wantSpecDef, err := ReadProto(c.wantCompiled, "")
				require.NoError(t, err)
				if !protobuf.Equal(wantSpecDef, protoStepDef) {
					t.Errorf("wanted:\n%+v\ngot:\n%+v", wantSpecDef, protoStepDef)
				}
			}
		})
	}

}
