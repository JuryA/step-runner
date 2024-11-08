package runner_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestStepFile(t *testing.T) {
	t.Run("step file is created empty", func(t *testing.T) {
		stepFile, err := runner.NewStepFileInDir(t.TempDir())
		require.NoError(t, err)

		file, err := os.Open(stepFile.Path())
		require.NoError(t, err)

		contents, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Len(t, contents, 0)
	})

	t.Run("read as step result", func(t *testing.T) {
		stepFile, err := runner.NewStepFileInDir(t.TempDir())
		require.NoError(t, err)

		file, err := os.OpenFile(stepFile.Path(), os.O_WRONLY, 0666)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		stepResult := bldr.StepResult().WithFailedStatus().Build()
		data, err := protojson.Marshal(stepResult)
		require.NoError(t, err)

		_, err = io.Copy(file, bytes.NewReader(data))
		require.NoError(t, err)

		loadedStepResult, err := stepFile.ReadStepResult()
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_failure, loadedStepResult.Status)
	})
}

func TestStepFile_ReadAsDotEnv(t *testing.T) {
	tests := map[string]struct {
		data    string
		want    map[string]string
		wantErr string
	}{
		"standard key/value": {
			data: `NAME=VALUE`,
			want: map[string]string{"NAME": "VALUE"},
		},
		"lowercase key/value": {
			data: `name=value`,
			want: map[string]string{"name": "value"},
		},
		"comments are stripped from lines": {
			data: `PROJECT TITLE=My Project # is this a comment`,
			want: map[string]string{"PROJECT TITLE": "My Project"},
		},
		"new lines can be added to value surrounded by quotes": {
			data: `KEY="one
two"`,
			want: map[string]string{"KEY": "one\ntwo"},
		},
		"new lines can be added to value surrounded by single quotes": {
			data: `KEY='one
two'`,
			want: map[string]string{"KEY": "one\ntwo"},
		},
		"unicode cannot be used in key": {
			data:    `spaß=German for fun`,
			wantErr: `unexpected character "\u009f" in variable name near "spaß=German for fun`,
		},
		"unicode can be used in value": {
			data: `FUN=spaß`,
			want: map[string]string{"FUN": "spaß"},
		},
		"keys can start with a number": {
			data: `2_MUCH_FUN=always`,
			want: map[string]string{"2_MUCH_FUN": "always"},
		},
		"empty space is removed": {
			data: `NAME=VALUE

NAME2=VALUE2

`,
			want: map[string]string{"NAME": "VALUE", "NAME2": "VALUE2"},
		},
		"keys and values are trimmed for space": {
			data: ` NAME =     VALUE      `,
			want: map[string]string{"NAME": "VALUE"},
		},
		"quotes can be added to value by surrounding with single quotes": {
			data: `KEY='"VALUE"'`,
			want: map[string]string{"KEY": `"VALUE"`},
		},
		"expressions can be added as the value": {
			data: `NAME=${{inputs.name}}`,
			want: map[string]string{"NAME": `${{inputs.name}}`},
		},
		"expressions cannot be added as the key": {
			data:    `${{inputs.name}}=Name value`,
			wantErr: `unexpected character "$" in variable name near "${{inputs.name}}=Name value"`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stepFile, err := runner.NewStepFileInDir(t.TempDir())
			require.NoError(t, err)

			file, err := os.OpenFile(stepFile.Path(), os.O_WRONLY, 0666)
			require.NoError(t, err)
			defer func() { _ = file.Close() }()

			_, err = io.Copy(file, strings.NewReader(test.data))
			require.NoError(t, err)

			env, err := stepFile.ReadDotEnv()

			if test.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, test.want, env)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.wantErr)
			}
		})
	}
}

func TestStepFile_ReadAsKeyValueLines(t *testing.T) {
	tests := map[string]struct {
		data    string
		want    map[string]string
		wantErr string
	}{
		"standard key/value": {
			data: `name=value`,
			want: map[string]string{"name": "value"},
		},
		"more than one equals": {
			data: `name=foo=bar`,
			want: map[string]string{"name": "foo=bar"},
		},
		"JSON string": {
			data: `message="hello"`,
			want: map[string]string{"message": `"hello"`},
		},
		"unicode can be used in key": {
			data: `spaß=German for fun`,
			want: map[string]string{"spaß": "German for fun"},
		},
		"unicode can be used in value": {
			data: `FUN=spaß`,
			want: map[string]string{"FUN": "spaß"},
		},
		"empty space is removed": {
			data: `name=value

name2=value2

`,
			want: map[string]string{"name": "value", "name2": "value2"},
		},
		"keys and values are not trimmed for space": {
			data: ` name =     value      `,
			want: map[string]string{" name ": "     value      "},
		},
		"expressions can be added as the value": {
			data: `name=${{inputs.name}}`,
			want: map[string]string{"name": `${{inputs.name}}`},
		},
		"expressions can be added as the key": {
			data: `${{inputs.name}}=Name value`,
			want: map[string]string{`${{inputs.name}}`: "Name value"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stepFile, err := runner.NewStepFileInDir(t.TempDir())
			require.NoError(t, err)

			file, err := os.OpenFile(stepFile.Path(), os.O_WRONLY, 0666)
			require.NoError(t, err)
			defer func() { _ = file.Close() }()

			_, err = io.Copy(file, strings.NewReader(test.data))
			require.NoError(t, err)

			env, err := stepFile.ReadKeyValueLines()

			if test.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, test.want, env)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.wantErr)
			}
		})
	}
}

func TestStepFile_ReadValues(t *testing.T) {
	cases := []struct {
		name              string
		outputMethod      proto.OutputMethod
		outputs           map[string]*proto.Spec_Content_Output
		writeToOutput     string
		wantOutput        map[string]*structpb.Value
		wantSubStepResult *proto.StepResult
		wantErr           string
	}{{
		name:       "no outputs",
		outputs:    map[string]*proto.Spec_Content_Output{},
		wantOutput: map[string]*structpb.Value{},
	}, {
		name: "single output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `value="foo"`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewStringValue("foo"),
		},
	}, {
		name: "multiple outputs",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
			"food":  {Type: proto.ValueType_string},
		},
		writeToOutput: "value=\"foo\"\nfood=\"apple\"",
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewStringValue("foo"),
			"food":  structpb.NewStringValue("apple"),
		},
	}, {
		name: "outputs with extra white space",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
			"food":  {Type: proto.ValueType_string},
		},
		writeToOutput: `

value="foo"

food="apple"

`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewStringValue("foo"),
			"food":  structpb.NewStringValue("apple"),
		},
	}, {
		name: "json string output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `value="foo"`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewStringValue("foo"),
		},
	}, {
		name: "json number output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_number},
		},
		writeToOutput: `value=12.34`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewNumberValue(12.34),
		},
	}, {
		name: "json bool output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_boolean},
		},
		writeToOutput: `value=true`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewBoolValue(true),
		},
	}, {
		name: "json empty struct output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_struct},
		},
		writeToOutput: `value={}`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{}}),
		},
	}, {
		name: "json full struct output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_struct},
		},
		writeToOutput: `value={"string":"bar","number":12.34,"bool":true,"null":null}`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
				"string": structpb.NewStringValue("bar"),
				"number": structpb.NewNumberValue(12.34),
				"bool":   structpb.NewBoolValue(true),
				"null":   structpb.NewNullValue(),
			}}),
		},
	}, {
		name: "json empty list output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_array},
		},
		writeToOutput: `value=[]`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{}}),
		},
	}, {
		name: "json full list output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_array},
		},
		writeToOutput: `value=["bar",12.34,true,null]`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("bar"),
				structpb.NewNumberValue(12.34),
				structpb.NewBoolValue(true),
				structpb.NewNullValue(),
			}}),
		},
	}, {
		name: "default output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {
				Type:    proto.ValueType_string,
				Default: structpb.NewStringValue("foo"),
			},
		},
		writeToOutput: ``,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewStringValue("foo"),
		},
	}, {
		name: "invalid format",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `invalid`,
		wantErr:       `reading outputs: invalid line "invalid"`,
	}, {
		name: "invalid json",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_struct},
		},
		writeToOutput: `value={foo}`,
		wantErr:       `output "value": malformed, unmarshaling json: invalid character 'f' looking for beginning of object key string`,
	}, {
		name: "missing output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
			"food":  {Type: proto.ValueType_string},
		},
		writeToOutput: `value="foo"`,
		wantErr:       `output "food": missing output, add to step outputs or remove from step specification`,
	}, {
		name: "extra output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
			"food":  {Type: proto.ValueType_string},
		},
		writeToOutput: `value="foo"
food="apple"
extra="output"`,
		wantErr: `output "extra": unexpected output, remove from step outputs or define in step specification`,
	}, {
		name: "wrong type received",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_number},
		},
		writeToOutput: `value="twelve"`,
		wantErr:       `output "value": mismatched types, declared as "number" in step specification and received from step as type "string"`,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outputStepFile, err := runner.NewStepFileInDir(t.TempDir())
			require.NoError(t, err)
			outputFile, err := os.OpenFile(outputStepFile.Path(), os.O_APPEND|os.O_WRONLY, 0660)
			require.NoError(t, err)
			_, err = outputFile.Write([]byte(tc.writeToOutput))
			require.NoError(t, err)
			err = outputFile.Close()
			require.NoError(t, err)

			outputs, err := outputStepFile.ReadValues(tc.outputs)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tc.wantOutput != nil {
				require.Equal(t, fmt.Sprintf("%v", tc.wantOutput), fmt.Sprintf("%v", outputs))
			}
		})
	}
}

func TestStepFile_ReadStepResult(t *testing.T) {
	cases := []struct {
		name              string
		writeToOutput     string
		wantSubStepResult *proto.StepResult
	}{{
		name:          "delegate output string",
		writeToOutput: `{"outputs":{"name":"steppy"}}`,
		wantSubStepResult: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"name": structpb.NewStringValue("steppy"),
			},
		},
	}, {
		name:          "delegate output struct",
		writeToOutput: `{"outputs":{"favorites":{"food":"hamburger"}}}`,
		wantSubStepResult: &proto.StepResult{
			Outputs: map[string]*structpb.Value{
				"favorites": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
					"food": structpb.NewStringValue("hamburger"),
				}}),
			},
		},
	},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outputStepFile, err := runner.NewStepFileInDir(t.TempDir())
			require.NoError(t, err)
			outputFile, err := os.OpenFile(outputStepFile.Path(), os.O_APPEND|os.O_WRONLY, 0660)
			require.NoError(t, err)
			_, err = outputFile.Write([]byte(tc.writeToOutput))
			require.NoError(t, err)
			err = outputFile.Close()
			require.NoError(t, err)

			delegate, err := outputStepFile.ReadStepResult()
			require.NoError(t, err)
			require.True(t, protobuf.Equal(tc.wantSubStepResult, delegate), "wanted %+v. got %+v", tc.wantSubStepResult, delegate)
		})
	}
}
