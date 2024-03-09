package step

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/proto"
	protobuf "google.golang.org/protobuf/proto"
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
---
type: steps
steps:
    - step:
          protocol: git
          url: "https://gitlab.com/components/script" # until we create the canonical step repository
          version: v1
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
---
type: steps
steps:
    - name: "my special script name"
      step:
          protocol: git
          url: "https://gitlab.com/components/script" # until we create the canonical step repository
          version: v1
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
            type: raw_string
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
      step: https://gitlab.com/components/foo#git@v1
    - inputs:
          greeting: ${{steps.foo to the max.outputs.greeting}}
      name: foo_redux
      step: ../steps/redux
`,
		wantCompiled: `
spec: {}
---
env:
    NAME: foo
type: steps
steps:
    - name: foo_to_the_max
      step:
          protocol: git
          url: "https://gitlab.com/components/foo" # until we create the canonical step repository
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

func TestReferenceCompiler(t *testing.T) {
	cases := []struct {
		ref     string
		want    *proto.Step_Reference
		wantErr bool
	}{{
		ref:     "invalid",
		wantErr: true,
	}, {
		ref:     "",
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
		ref: "gitlab.com/components/script/#git@v1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script/",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {
		ref: "https://gitlab.com/components/script/#git@v1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script/",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {

		ref: "gitlab.com/components/script/#bash,git@v1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script/",
			Path:     []string{"bash"},
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {
		ref: "gitlab.com/components/script/#bash/my-step.yml,git@v1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script/",
			Path:     []string{"bash"},
			Filename: "my-step.yml",
			Version:  "v1",
		},
	}, {
		ref: "http://bad.idea.com/my-step/#git@v1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "http://bad.idea.com/my-step/",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {
		ref: "gitlab.com/components/script/#git@v2.1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script/",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v2.1",
		},
	}, {
		ref: "gitlab.com/components/script/#git@20e9c40c",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script/",
			Path:     nil,
			Filename: "step.yml",
			Version:  "20e9c40c",
		},
	}, {
		ref: "gitlab.com/components/script/#git@20e9c40c9213f2a044e4a81906956a779af3da4b",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/components/script/",
			Path:     nil,
			Filename: "step.yml",
			Version:  "20e9c40c9213f2a044e4a81906956a779af3da4b",
		},
	}, {
		ref: "registry.gitlab.com/components/script/#oci@latest",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_oci,
			Url:      "https://registry.gitlab.com/components/script/",
			Path:     nil,
			Filename: "step.yml",
			Version:  "latest",
		},
	}, {
		ref: "registry.gitlab.com/components/script/#oci@v1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_oci,
			Url:      "https://registry.gitlab.com/components/script/",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {
		ref: "registry.gitlab.com/components/script/#oci@sha256:83c876be7b35b6c5c892b1347ee894f23b26e00a686e3cbb51b004263c014f8c",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_oci,
			Url:      "https://registry.gitlab.com/components/script/",
			Path:     nil,
			Filename: "step.yml",
			Version:  "sha256:83c876be7b35b6c5c892b1347ee894f23b26e00a686e3cbb51b004263c014f8c",
		},
	}, {
		ref: "registry.gitlab.com/components/script/#bash,oci@v1",
		want: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_oci,
			Url:      "https://registry.gitlab.com/components/script/",
			Path:     []string{"bash"},
			Filename: "step.yml",
			Version:  "v1",
		},
	}, {
		ref:     "ftp://gitlab.com/components/script/#git@v1", // unsupported
		wantErr: true,
	}, {
		ref:     "notavalidscheme://gitlab.com/components/script/#git@v1",
		wantErr: true,
	}}

	for _, c := range cases {
		t.Run(c.ref, func(t *testing.T) {
			got, err := (*referenceCompiler)(&c.ref).compile()
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, protobuf.Equal(c.want, got), "want %v. got %v", c.want, got)
			}
		})
	}
}
