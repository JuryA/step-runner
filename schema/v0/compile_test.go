package schema

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name         string
		steps        string
		wantCompiled string
		wantErr      bool
	}{{
		name: "spec is optional",
		steps: `
{}
---
steps:
- script: echo hello world
`,
		wantCompiled: `
spec:
    inputs: {}
    output_method: "outputs"
---
type: steps
steps:
    - name: "0"
      step:
          protocol: git
          url: "https://gitlab.com/components/script"
          version: main
          filename: step.yml
      inputs:
          script: echo hello world
`,
	}, {
		name: "simple case",
		steps: `
spec:
    inputs:
        name:
---
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
    output_method: "outputs"
---
type: exec
exec:
    command:
        - echo
        - ${{inputs.name}}
`,
	}, {
		name: "step script keyword compiles to a single step",
		steps: `
spec:
---
steps:
  - name: "my special script name"
    script: echo hello world
`,
		wantCompiled: `
spec:
    inputs: {}
    output_method: "outputs"
---
type: steps
steps:
    - name: "my special script name"
      step:
          protocol: git
          url: "https://gitlab.com/components/script"
          version: main
          filename: step.yml
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
            type: boolean
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
            type: boolean
        name:
            default: steppy
            type: string
    outputs:
        eye_color:
            type: raw_string
            default: brown
    output_method: "outputs"
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
env:
    NAME: foo
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
      step: https://gitlab.com/components/foo@v1
    - inputs:
          greeting: ${{steps.foo to the max.outputs.greeting}}
      name: foo_redux
      step: ../steps/redux
`,
		wantCompiled: `
spec:
    output_method: "outputs"
---
env:
    NAME: foo
type: steps
steps:
    - name: foo_to_the_max
      step:
          protocol: git
          url: "https://gitlab.com/components/foo"
          version: v1
          filename: step.yml
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
      step:
          protocol: local
          path: [ '..', steps, redux ]
          filename: step.yml
      inputs:
          greeting: ${{steps.foo to the max.outputs.greeting}}
outputs:
    eye_color: brown
`,
	}, {
		name: "compile action keyword to step",
		steps: `
spec: {}
---
steps:
    - name: find_something
      action: mikefarah/yq@master
      inputs:
          cmd: yq .name some.yaml
`,
		wantCompiled: `
spec:
    output_method: "outputs"
---
type: steps
steps:
    - name: find_something
      step:
          protocol: git
          url: "https://gitlab.com/components/action-runner"
          version: main
          filename: step.yml
      inputs:
          action: mikefarah/yq@master
          inputs:
              cmd: yq .name some.yaml
`,
	}, {
		name: "compile delegate output method",
		steps: `
spec:
  outputs: delegate
---
steps:
    - name: delegate_me
      step: https://gitlab.com/components/foo@v1
delegate: delegate_me
`,
		wantCompiled: `
spec:
    output_method: delegate
---
type: steps
steps:
    - name: delegate_me
      step:
          protocol: git
          url: "https://gitlab.com/components/foo"
          version: v1
          filename: step.yml
delegate: delegate_me
`,
	}, {
		name: "name is optional",
		steps: `
spec: {}
---
steps:
    - step: ./one
    - step: ./two
`,
		wantCompiled: `
spec:
    output_method: outputs
---
type: steps
steps:
    - name: "0"
      step:
          protocol: local
          path: [ ., one ]
          filename: step.yml
    - name: "1"
      step:
          protocol: local
          path: [ ., two ]
          filename: step.yml
`,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec, step, err := ReadSteps(c.steps, "")
			require.NoError(t, err)
			protoSpec, err := spec.Compile()
			require.NoError(t, err)
			protoDef, err := step.compileDefinition()
			require.NoError(t, err)
			protoSpecDef := &proto.SpecDefinition{
				Spec:       protoSpec,
				Definition: protoDef,
			}
			if c.wantErr {
				require.Error(t, err)
				require.Nil(t, protoSpecDef)
			} else {
				require.NoError(t, err)
				wantSpecDef, err := readProto(c.wantCompiled, "")
				require.NoError(t, err)
				if !protobuf.Equal(wantSpecDef, protoSpecDef) {
					t.Errorf("wanted:\n%+v\ngot:\n%+v", wantSpecDef, protoSpecDef)
				}
			}
		})
	}

}

func TestReferenceCompiler(t *testing.T) {
	cases := []struct {
		ref     string
		want    *proto.Step_Reference
		wantErr bool
	}{{
		ref:     "invalid",
		wantErr: true,
	}, {
		ref: ".",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_local,
			Url:      "",
			Path:     []string{"."},
			Filename: "step.yml",
			Version:  "",
		},
	}, {
		ref: "./path/to/my/file",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_local,
			Url:      "",
			Path:     []string{".", "path", "to", "my", "file"},
			Filename: "step.yml",
			Version:  "",
		},
	}, {
		ref: "gitlab.com/components/script@v1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {
		ref: "https://gitlab.com/components/script@v1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {
		ref: `
git:
    url:     gitlab.com/components/script
    dir:     bash
    rev:  v1
`,
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script",
			Path:     []string{"bash"},
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {
		ref: `
git:
    url:    http://bad.idea.com/my-step
    rev: v1
`,
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "http://bad.idea.com/my-step",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {
		ref: `
git:
    url:    gitlab.com/components/script
    rev: v2.1
`,
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v2.1",
		},
	}, {
		ref: `
git:
    url:    gitlab.com/components/script
    rev: 20e9c40c
`,
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script",
			Path:     nil,
			Filename: "step.yml",
			Version:  "20e9c40c",
		},
	}, {
		ref: `
git:
    url:    gitlab.com/components/script
    rev: 20e9c40c9213f2a044e4a81906956a779af3da4b
`,
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script",
			Path:     nil,
			Filename: "step.yml",
			Version:  "20e9c40c9213f2a044e4a81906956a779af3da4b",
		},
	}, {
		ref:     "ftp://gitlab.com/components/script@v1", // unsupported
		wantErr: true,
	}, {
		ref:     "notavalidscheme://gitlab.com/components/script@v1",
		wantErr: true,
	}}

	for _, c := range cases {
		t.Run(c.ref, func(t *testing.T) {
			stepStr := fmt.Sprintf("-step: %s", c.ref)
			step := &Step{}
			err := unmarshalSchema(stepStr, step)
			require.NoError(t, err)
			got, err := step.CompileStep(0)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				require.True(t, protobuf.Equal(c.want, got.Step), "want %v. got %v", c.want, got.Step)
			}
		})
	}
}
