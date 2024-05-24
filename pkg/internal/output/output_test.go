package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestNew(t *testing.T) {
	// New creates output and export files for each step call.  It
	// also provides the directory as `${{ call_dir }}` so steps
	// can write temporary files relevant to a given step
	// invokation.  The output and export files must be in the
	// call directory.
	globalCtx := context.NewGlobal()
	stepCtx := context.NewSteps(globalCtx)
	files, err := New(stepCtx, nil)
	require.NoError(t, err)
	callDir := files.stepCtx.CallDir
	require.NotEmpty(t, callDir)
	outputFilename := files.outputFile
	require.NotEmpty(t, outputFilename)
	exportFilename := files.exportFile
	require.NotEmpty(t, exportFilename)
	require.True(t, strings.HasPrefix(outputFilename, callDir))
	require.True(t, strings.HasPrefix(exportFilename, callDir))
}

func TestOutput(t *testing.T) {
	cases := []struct {
		name          string
		outputs       map[string]*proto.Spec_Content_Output
		writeToOutput string
		want          *proto.StepResult
		wantErr       bool
	}{{
		name:    "no outputs",
		outputs: map[string]*proto.Spec_Content_Output{},
		want:    &proto.StepResult{},
	}, {
		name: "single output",
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
		name: "multiple outputs",
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
		name: "outputs with extra white space",
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
		name: "json string output",
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
		name: "json number output",
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
		name: "json bool output",
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
		name: "json empty struct output",
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
		name: "json full struct output",
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
		name: "json empty list output",
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
		name: "json full list output",
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
		name: "default output",
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
		name: "invalid format",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_raw_string},
		},
		writeToOutput: `invalid`,
		wantErr:       true,
	}, {
		name: "invalid json",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `value=foo`,
		wantErr:       true,
	}, {
		name: "missing output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_raw_string},
			"food":  {Type: proto.ValueType_raw_string},
		},
		writeToOutput: "value=foo",
		wantErr:       true,
	}, {
		name: "extra output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_raw_string},
			"food":  {Type: proto.ValueType_raw_string},
		},
		writeToOutput: "value=foo\nfood=apple\nextra=output",
		wantErr:       true,
	}, {
		name: "wrong type received",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `value=12.34`,
		wantErr:       true,
	}, {
		name: "delegate output string",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_step_result},
		},
		writeToOutput: `value={"outputs":{"name":"steppy"}}`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"name": structpb.NewStringValue("steppy"),
			},
			ChildrenStepResults: []*proto.StepResult{{
				Outputs: map[string]*structpb.Value{
					"name": structpb.NewStringValue("steppy"),
				},
			}},
		},
	}, {
		name: "delegate output struct",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_step_result},
		},
		writeToOutput: `value={"outputs":{"favorites":{"food":"hamburger"}}}`,
		want: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"favorites": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
					"food": structpb.NewStringValue("hamburger"),
				}}),
			},
			ChildrenStepResults: []*proto.StepResult{{
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
			files, err := New(context.NewSteps(context.NewGlobal()), tc.outputs)
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

}
