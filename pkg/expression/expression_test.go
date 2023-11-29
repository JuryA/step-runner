package expression

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestInterpolateInputs(t *testing.T) {
	cases := []struct {
		name      string
		globalCtx *context.Global
		stepsCtx  *context.Steps
		step      *proto.Step
		wantStep  *proto.Step
		wantErr   bool
	}{{
		name: "simple case everything",
		globalCtx: &context.Global{
			Job: map[string]string{
				"JOB_ID": "1234",
			},
			Env: map[string]string{
				"USER": "steppy",
			},
		},
		stepsCtx: &context.Steps{
			Outputs: map[string]map[string]string{
				"foo": {
					"bar": "baz",
				},
			},
		},
		step: &proto.Step{
			Name: "step",
			Step: "uri",
			Env: map[string]string{
				"job-id": "${{job.JOB_ID}}",
			},
			Inputs: map[string]*structpb.Value{
				"user":    structpb.NewStringValue("${{env.USER}}"),
				"foo-bar": structpb.NewStringValue("${{steps.foo.outputs.bar}}"),
			},
		},
		wantStep: &proto.Step{
			Name: "step",
			Step: "uri",
			Env: map[string]string{
				"job-id": "1234",
			},
			Inputs: map[string]*structpb.Value{
				"user":    structpb.NewStringValue("steppy"),
				"foo-bar": structpb.NewStringValue("baz"),
			},
		},
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := InterpolateInputs(c.globalCtx, c.stepsCtx, c.step)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if !protobuf.Equal(c.wantStep, c.step) {
					t.Errorf("wanted:\n%+v\ngot:\n%+v", c.wantStep, c.step)
				}
			}
		})
	}
}

func TestInterpolateOutputs(t *testing.T) {
	cases := []struct {
		name        string
		globalCtx   *context.Global
		stepsCtx    *context.Steps
		spec        *proto.Spec_Content
		def         *proto.Definition
		wantOutputs map[string]string
		wantErr     bool
	}{{
		name: "default output",
		spec: &proto.Spec_Content{
			Outputs: map[string]*proto.Spec_Content_Output{
				"output": {
					Default: "value",
				},
			},
		},
		def: &proto.Definition{},
		wantOutputs: map[string]string{
			"output": "value",
		},
	}, {
		name: "undeclared output",
		spec: &proto.Spec_Content{},
		def: &proto.Definition{
			Outputs: map[string]string{
				"output": "value",
			},
		},
		wantErr: true,
	}, {
		name: "output literal",
		spec: &proto.Spec_Content{
			Outputs: map[string]*proto.Spec_Content_Output{
				"output": {},
			},
		},
		def: &proto.Definition{
			Outputs: map[string]string{
				"output": "value",
			},
		},
		wantOutputs: map[string]string{
			"output": "value",
		},
	}, {
		name: "global environment and job variables",
		globalCtx: &context.Global{
			Env: map[string]string{
				"FOO": "bar",
			},
			Job: map[string]string{
				"id": "1234",
			},
		},
		spec: &proto.Spec_Content{
			Outputs: map[string]*proto.Spec_Content_Output{
				"output": {},
			},
		},
		def: &proto.Definition{
			Outputs: map[string]string{
				"output": "${{env.FOO}} ${{job.id}}",
			},
		},
		wantOutputs: map[string]string{
			"output": "bar 1234",
		},
	}, {
		name: "step outputs",
		stepsCtx: &context.Steps{
			Outputs: map[string]map[string]string{
				"step-1": {
					"output-1": "value-1",
				},
			},
		},
		spec: &proto.Spec_Content{
			Outputs: map[string]*proto.Spec_Content_Output{
				"output": {},
			},
		},
		def: &proto.Definition{
			Outputs: map[string]string{
				"output": "${{steps.step-1.outputs.output-1}}",
			},
		},
		wantOutputs: map[string]string{
			"output": "value-1",
		},
	}, {
		name: "do not interpolate defaults (or anything in spec)",
		globalCtx: &context.Global{
			Env: map[string]string{
				"FOO": "bar",
			},
		},
		spec: &proto.Spec_Content{
			Outputs: map[string]*proto.Spec_Content_Output{
				"output": {
					Default: "${{env.FOO}}",
				},
			},
		},
		def: &proto.Definition{},
		wantOutputs: map[string]string{
			"output": "${{env.FOO}}",
		},
	}, {
		name: "a little of everything",
		globalCtx: &context.Global{
			Env: map[string]string{
				"env-1": "A",
			},
			Job: map[string]string{
				"job-1": "B",
			},
		},
		stepsCtx: &context.Steps{
			Outputs: map[string]map[string]string{
				"step-1": {
					"output-1": "C",
					"output-2": "D",
				},
				"step-2": {
					"output-1": "E",
				},
			},
		},
		spec: &proto.Spec_Content{
			Outputs: map[string]*proto.Spec_Content_Output{
				"output": {},
			},
		},
		def: &proto.Definition{
			Outputs: map[string]string{
				"output": `
${{env.env-1}}
${{job.job-1}}
${{steps.step-1.outputs.output-1}}
${{steps.step-1.outputs.output-2}}
${{steps.step-2.outputs.output-1}}
`,
			},
		},
		wantOutputs: map[string]string{
			"output": `
A
B
C
D
E
`,
		},
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			outputs, err := InterpolateOutputs(c.globalCtx, c.stepsCtx, c.spec, c.def)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, c.wantOutputs, outputs)
			}
		})
	}
}

func TestInterpolateExec(t *testing.T) {
	cases := []struct {
		name     string
		inputs   map[string]*structpb.Value
		spec     *proto.Spec_Content
		exec     *proto.Definition_Exec
		wantExec *proto.Definition_Exec
		wantErr  bool
	}{{
		name: "simple case",
		inputs: map[string]*structpb.Value{
			"foo": structpb.NewStringValue("1234"),
		},
		spec: &proto.Spec_Content{
			Inputs: map[string]*proto.Spec_Content_Input{
				"foo": {
					Type: proto.InputType_string,
				},
			},
		},
		exec: &proto.Definition_Exec{
			Command: []string{
				"${{inputs.foo}}",
			},
		},
		wantExec: &proto.Definition_Exec{
			Command: []string{
				"1234",
			},
		},
	}, {
		name: "wrong type",
		inputs: map[string]*structpb.Value{
			"foo": structpb.NewStringValue("1234"),
		},
		spec: &proto.Spec_Content{
			Inputs: map[string]*proto.Spec_Content_Input{
				"foo": {
					Type: proto.InputType_number,
				},
			},
		},
		exec: &proto.Definition_Exec{
			Command: []string{
				"${{inputs.foo}}",
			},
		},
		wantErr: true,
	}, {
		name: "all types with defaults",
		inputs: map[string]*structpb.Value{
			"foo": structpb.NewStringValue("steppy"),
			"bar": structpb.NewNumberValue(1),
			"baz": structpb.NewBoolValue(true),
			"bam": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"umm": structpb.NewListValue(&structpb.ListValue{}),
				},
			}),
		},
		spec: &proto.Spec_Content{
			Inputs: map[string]*proto.Spec_Content_Input{
				"foo": {
					Type: proto.InputType_string,
				},
				"bar": {
					Type: proto.InputType_number,
				},
				"baz": {
					Type: proto.InputType_bool,
				},
				"bam": {
					Type: proto.InputType_struct,
				},
			},
		},
		exec: &proto.Definition_Exec{
			Command: []string{
				"${{inputs.foo}}",
				"${{inputs.bar}}",
				"${{inputs.baz}}",
				"${{inputs.bam}}",
			},
		},
		wantExec: &proto.Definition_Exec{
			Command: []string{
				"steppy",
				"1",
				"true",
				`{"umm":[]}`,
			},
		},
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := InterpolateExec(context.NewGlobal(), c.inputs, c.spec, c.exec)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if !protobuf.Equal(c.wantExec, c.exec) {
					t.Errorf("wanted:\n%+v\ngot:\n%+v", c.wantExec, c.exec)
				}
			}
		})
	}
}
