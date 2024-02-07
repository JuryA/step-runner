package step

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/proto"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		wantSpec *proto.Spec
		wantDef  *proto.Definition
		wantErr  bool
	}{{
		name: "simple case",
		yaml: `
spec:
    inputs:
        name:
---
exec:
    command:
        - echo
        - ${{inputs.name}}
type: exec
`,
		wantSpec: &proto.Spec{
			Spec: &proto.Spec_Content{
				Inputs: map[string]*proto.Spec_Content_Input{
					"name": {
						Type: proto.InputType_string,
					},
				},
			},
		},
		wantDef: &proto.Definition{
			Type: proto.DefinitionType_exec,
			Exec: &proto.Definition_Exec{
				Command: []string{
					"echo",
					"${{inputs.name}}",
				},
			},
		},
	}, {
		name: "complex type: exec",
		yaml: `
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
		wantSpec: &proto.Spec{
			Spec: &proto.Spec_Content{
				Inputs: map[string]*proto.Spec_Content_Input{
					"name": {
						Type:    proto.InputType_string,
						Default: structpb.NewStringValue("steppy"),
					},
					"age": {
						Type:    proto.InputType_number,
						Default: structpb.NewNumberValue(1),
					},
					"hungry": {
						Type:    proto.InputType_bool,
						Default: structpb.NewBoolValue(false),
					},
					"favorites": {
						Type: proto.InputType_struct,
						Default: structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"color": structpb.NewStringValue("red"),
							},
						}),
					},
				},
				Outputs: map[string]*proto.Spec_Content_Output{
					"eye_color": {
						Default: "brown",
					},
				},
			},
		},
		wantDef: &proto.Definition{
			Type: proto.DefinitionType_exec,
			Exec: &proto.Definition_Exec{
				Command: []string{
					"echo",
					"meet ${{inputs.name}}",
					"who is ${{inputs.age}}",
					"likes ${{inputs.favorites}}",
					"and is hungry (${{inputs.hungry}}).",
				},
			},
		},
	}, {
		name: "complex type: steps",
		yaml: `
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
		wantSpec: &proto.Spec{Spec: &proto.Spec_Content{}},
		wantDef: &proto.Definition{
			Type: proto.DefinitionType_steps,
			Steps: []*proto.Step{{
				Name: "foo_to_the_max",
				Step: "git+https://gitlab.com/gitlab-org/foo@v1",
				Env: map[string]string{
					"USER":   "srunner",
					"JOB_ID": "${{job.id}}",
				},
				Inputs: map[string]*structpb.Value{
					"age": structpb.NewNumberValue(1),
					"favorites": structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"food": structpb.NewListValue(&structpb.ListValue{
								Values: []*structpb.Value{
									structpb.NewStringValue("hamburger"),
									structpb.NewStringValue("sausage"),
								},
							}),
						},
					}),
					"hungry": structpb.NewBoolValue(false),
					"name":   structpb.NewStringValue("steppy"),
				},
			}, {
				Name: "foo_redux",
				Step: "../steps/redux",
				Inputs: map[string]*structpb.Value{
					"greeting": structpb.NewStringValue("${{steps.foo to the max.outputs.greeting}}"),
				},
			}},
			Outputs: map[string]string{
				"eye_color": "brown",
			},
		},
	}, {
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
		name: "missing exec",
		yaml: `
spec:
  inputs:
    name:
`,
		wantErr: true,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stepDef, err := Compile(c.yaml, "")
			if c.wantErr {
				require.Error(t, err)
				require.Nil(t, stepDef)
			} else {
				require.NoError(t, err)
				if !protobuf.Equal(c.wantSpec, stepDef.Spec) {
					t.Errorf("wanted:\n%+v\ngot:\n%+v", c.wantSpec, stepDef.Spec)
				}
				if !protobuf.Equal(c.wantDef, stepDef.Definition) {
					t.Errorf("wanted:\n%+v\ngot:\n%+v", c.wantDef, stepDef.Definition)
				}
			}
		})
	}
}
