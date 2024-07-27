package output

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/domain"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestOutput(t *testing.T) {
	cases := []struct {
		name          string
		outputMethod  proto.OutputMethod
		outputs       map[string]*proto.Spec_Content_Output
		writeToOutput string
		want          *proto.StepResult
		wantErr       bool
	}{{
		name:         "no outputs",
		outputMethod: proto.OutputMethod_outputs,
		outputs:      map[string]*proto.Spec_Content_Output{},
		want:         &proto.StepResult{},
	}, {
		name:         "single output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_raw_string},
		},
		writeToOutput: `value=foo`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewStringValue("foo"),
			},
		},
	}, {
		name:         "multiple outputs",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_raw_string},
			"food":  {Type: proto.ValueType_raw_string},
		},
		writeToOutput: "value=foo\nfood=apple",
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewStringValue("foo"),
				"food":  structpb.NewStringValue("apple"),
			},
		},
	}, {
		name:         "outputs with extra white space",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_raw_string},
			"food":  {Type: proto.ValueType_raw_string},
		},
		writeToOutput: `

value=foo

food=apple

`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewStringValue("foo"),
				"food":  structpb.NewStringValue("apple"),
			},
		},
	}, {
		name:         "json string output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `value="foo"`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewStringValue("foo"),
			},
		},
	}, {
		name:         "json number output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_number},
		},
		writeToOutput: `value=12.34`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewNumberValue(12.34),
			},
		},
	}, {
		name:         "json bool output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_boolean},
		},
		writeToOutput: `value=true`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewBoolValue(true),
			},
		},
	}, {
		name:         "json empty struct output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_struct},
		},
		writeToOutput: `value={}`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{}}),
			},
		},
	}, {
		name:         "json full struct output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_struct},
		},
		writeToOutput: `value={"string":"bar","number":12.34,"bool":true,"null":null}`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
					"string": structpb.NewStringValue("bar"),
					"number": structpb.NewNumberValue(12.34),
					"bool":   structpb.NewBoolValue(true),
					"null":   structpb.NewNullValue(),
				}}),
			},
		},
	}, {
		name:         "json empty list output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_array},
		},
		writeToOutput: `value=[]`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{}}),
			},
		},
	}, {
		name:         "json full list output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_array},
		},
		writeToOutput: `value=["bar",12.34,true,null]`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
					structpb.NewStringValue("bar"),
					structpb.NewNumberValue(12.34),
					structpb.NewBoolValue(true),
					structpb.NewNullValue(),
				}}),
			},
		},
	}, {
		name:         "default output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {
				Type:    proto.ValueType_string,
				Default: structpb.NewStringValue("foo"),
			},
		},
		// No output written
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"value": structpb.NewStringValue("foo"),
			},
		},
	}, {
		name:         "invalid format",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_raw_string},
		},
		writeToOutput: `invalid`,
		wantErr:       true,
	}, {
		name:         "invalid json",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `value=foo`,
		wantErr:       true,
	}, {
		name:         "missing output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_raw_string},
			"food":  {Type: proto.ValueType_raw_string},
		},
		writeToOutput: "value=foo",
		wantErr:       true,
	}, {
		name:         "extra output",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_raw_string},
			"food":  {Type: proto.ValueType_raw_string},
		},
		writeToOutput: "value=foo\nfood=apple\nextra=output",
		wantErr:       true,
	}, {
		name:         "wrong type received",
		outputMethod: proto.OutputMethod_outputs,
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `value=12.34`,
		wantErr:       true,
	}, {
		name:          "delegate output string",
		outputMethod:  proto.OutputMethod_delegate,
		outputs:       nil,
		writeToOutput: `{"outputs":{"name":"steppy"}}`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"name": structpb.NewStringValue("steppy"),
			},
			SubStepResults: []*proto.StepResult{{
				Outputs: map[string]*structpb.Value{
					"name": structpb.NewStringValue("steppy"),
				},
			}},
		},
	}, {
		name:          "delegate output struct",
		outputMethod:  proto.OutputMethod_delegate,
		outputs:       nil,
		writeToOutput: `{"outputs":{"favorites":{"food":"hamburger"}}}`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"favorites": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
					"food": structpb.NewStringValue("hamburger"),
				}}),
			},
			SubStepResults: []*proto.StepResult{{
				Outputs: map[string]*structpb.Value{
					"favorites": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
						"food": structpb.NewStringValue("hamburger"),
					}}),
				},
			}},
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := domain.NewGlobalCtx()
			require.NoError(t, err)
			defer ctx.Cleanup()
			files, err := New(domain.NewStepsCtx(ctx), tc.outputMethod, tc.outputs)
			require.NoError(t, err)

			outputFile, err := os.OpenFile(filepath.Join(files.dir, outputFilename), os.O_APPEND|os.O_WRONLY, 0660)
			require.NoError(t, err)
			_, err = outputFile.Write([]byte(tc.writeToOutput))
			require.NoError(t, err)
			err = outputFile.Close()
			require.NoError(t, err)

			got := &proto.StepResult{}
			err = files.OutputTo(got)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, protobuf.Equal(tc.want, got), "wanted %+v. got %+v", tc.want, got)
			}
		})
	}
}

func TestExport(t *testing.T) {
	cases := []struct {
		name          string
		globalEnv     map[string]string
		writeToExport string
		wantExports   map[string]string
		wantGlobalEnv map[string]string
	}{{
		name: "no export",
	}, {
		name: "no export keeping global env",
		globalEnv: map[string]string{
			"foo": "bar",
		},
		wantGlobalEnv: map[string]string{
			"foo": "bar",
		},
	}, {
		name: "export overwriting global env",
		globalEnv: map[string]string{
			"foo": "bar",
		},
		writeToExport: "foo=baz",
		wantExports: map[string]string{
			"foo": "baz",
		},
		wantGlobalEnv: map[string]string{
			"foo": "baz",
		},
	}, {
		name: "export multiple times last value controls",
		writeToExport: `
foo=bar
foo=baz
`,
		wantExports: map[string]string{
			"foo": "baz",
		},
		wantGlobalEnv: map[string]string{
			"foo": "baz",
		},
	}, {
		name: "re-export a value",
		globalEnv: map[string]string{
			"foo": "bar",
		},
		writeToExport: "foo=bar",
		wantExports: map[string]string{
			"foo": "bar",
		},
		wantGlobalEnv: map[string]string{
			"foo": "bar",
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := domain.NewGlobalCtx()
			require.NoError(t, err)
			if tc.globalEnv != nil {
				ctx.Env = tc.globalEnv
			}
			defer ctx.Cleanup()

			exportFile, err := os.OpenFile(filepath.Join(ctx.ExportFile), os.O_APPEND|os.O_WRONLY, 0660)
			require.NoError(t, err)
			_, err = exportFile.Write([]byte(tc.writeToExport))
			require.NoError(t, err)
			err = exportFile.Close()
			require.NoError(t, err)

			got := &proto.StepResult{}
			err = ctx.ExportTo(got)
			require.NoError(t, err)
			require.True(t, maps.Equal(tc.wantExports, got.Exports), "want %+v. got %+v", tc.wantExports, got.Exports)
			require.True(t, maps.Equal(tc.wantGlobalEnv, ctx.Env), "want %+v. got %+v", tc.wantGlobalEnv, ctx.Env)
		})
	}
}
