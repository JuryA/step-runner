package runner_test

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
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
}

func TestStepFile_ReadEnvironment(t *testing.T) {
	tests := map[string]struct {
		data    string
		want    map[string]string
		wantErr string
	}{
		"value as string": {
			data: `{"name":"NAME","value":"VALUE"}`,
			want: map[string]string{"NAME": "VALUE"},
		},
		"value as number": {
			data: `{"name":"NAME","value":56.99}`,
			want: map[string]string{"NAME": "56.99"},
		},
		"value as bool": {
			data: `{"name":"NAME","value":false}`,
			want: map[string]string{"NAME": "false"},
		},
		"value as null": {
			data: `{"name":"NAME","value":""}`,
			want: map[string]string{"NAME": ""},
		},
		"value as list": {
			data:    `{"name":"NAME","value":[1,2,3]}`,
			wantErr: `read env file: key "NAME": cannot convert value type "array" to string`,
		},
		"value as struct": {
			data:    `{"name":"NAME","value":{"value":"ah-oh"}}`,
			wantErr: `read env file: key "NAME": cannot convert value type "struct" to string`,
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

			env, err := stepFile.ReadEnvironment()

			if test.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, test.want, env.Values())
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.wantErr)
			}
		})
	}
}

func TestStepFile_ReadValues(t *testing.T) {
	cases := []struct {
		name          string
		debugMode     bool
		outputs       map[string]*proto.Spec_Content_Output
		writeToOutput string
		wantOutput    map[string]*structpb.Value
		wantErr       string
	}{{
		name:       "no outputs",
		outputs:    map[string]*proto.Spec_Content_Output{},
		wantOutput: map[string]*structpb.Value{},
	}, {
		name: "multiple outputs",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
			"food":  {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"value","value":"foo"}
{"name":"food","value":"apple"}`,
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

{  "name":"value" ,     "value":"foo"  }

       {"name":"food","value":"apple"}     

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
		writeToOutput: `{"name":"value","value":"foo"}`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewStringValue("foo"),
		},
	}, {
		name: "json number output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_number},
		},
		writeToOutput: `{"name":"value","value":12.34}`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewNumberValue(12.34),
		},
	}, {
		name: "json bool output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_boolean},
		},
		writeToOutput: `{"name":"value","value":true}`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewBoolValue(true),
		},
	}, {
		name: "json empty struct output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_struct},
		},
		writeToOutput: `{"name":"value","value":{}}`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{}}),
		},
	}, {
		name: "json full struct output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_struct},
		},
		writeToOutput: `{"name":"value","value":{"string":"bar","number":12.34,"bool":true,"null":null}}`,
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
		writeToOutput: `{"name":"value","value":[]}`,
		wantOutput: map[string]*structpb.Value{
			"value": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{}}),
		},
	}, {
		name: "json full list output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_array},
		},
		writeToOutput: `{"name":"value","value":["bar",12.34,true,null]}`,
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
		name: "keys and values are not trimmed for space",
		outputs: map[string]*proto.Spec_Content_Output{
			"  value  ": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"  value  ","value":"   foo   "}`,
		wantOutput: map[string]*structpb.Value{
			"  value  ": structpb.NewStringValue("   foo   "),
		},
	}, {
		name: "new lines are preserved in keys and values",
		outputs: map[string]*proto.Spec_Content_Output{
			`na\nme`: {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"na\\nme","value":"fo\\noo"}`,
		wantOutput: map[string]*structpb.Value{
			`na\nme`: structpb.NewStringValue(`fo\noo`),
		},
	}, {
		name: "expression can be used in key",
		outputs: map[string]*proto.Spec_Content_Output{
			"${{inputs.name}}": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"${{inputs.name}}","value":"foo"}`,
		wantOutput: map[string]*structpb.Value{
			"${{inputs.name}}": structpb.NewStringValue("foo"),
		},
	}, {
		name: "expression can be used in value",
		outputs: map[string]*proto.Spec_Content_Output{
			"name": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"name","value":"${{inputs.value}}"}`,
		wantOutput: map[string]*structpb.Value{
			"name": structpb.NewStringValue("${{inputs.value}}"),
		},
	}, {
		name: "unicode can be used in key",
		outputs: map[string]*proto.Spec_Content_Output{
			"spaß": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"spaß","value":"German for fun"}`,
		wantOutput: map[string]*structpb.Value{
			"spaß": structpb.NewStringValue("German for fun"),
		},
	}, {
		name: "unicode can be used in value",
		outputs: map[string]*proto.Spec_Content_Output{
			"FUN": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"FUN","value":"spaß"}`,
		wantOutput: map[string]*structpb.Value{
			"FUN": structpb.NewStringValue("spaß"),
		},
	}, {
		name: "invalid format",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `invalid`,
		wantErr:       `read output file: failed to unmarshal JSON at line 1: invalid character 'i' looking for beginning of value`,
	}, {
		name: "previous key JSON value format",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `value="foo"`,
		wantErr:       `read output file: failed to unmarshal JSON at line 1: invalid character 'v' looking for beginning of value`,
	}, {
		name: "invalid json",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name"::"value","value":"foo"}`,
		wantErr:       `read output file: failed to unmarshal JSON at line 1: invalid character ':' looking for beginning of value`,
	}, {
		name: "missing output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
			"food":  {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"value","value":"foo"}`,
		wantErr:       `read output file: key "food": missing output, add to step outputs or remove from step specification`,
	}, {
		name: "extra output",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
			"food":  {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"value","value":"foo"}
{"name":"food","value":"apple"}
{"name":"extra","value":"output"}`,
		wantErr: `read output file: key "extra": unexpected output, remove from step outputs or define in step specification`,
	}, {
		name: "wrong type received",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_number},
		},
		writeToOutput: `{"name":"value","value":"twelve"}`,
		wantErr:       `read output file: key "value": mismatched types, declared as "number" in step specification and received from step as type "string"`,
	}, {
		name: "missing name key",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"value":"foo"}`,
		wantErr:       `read output file: failed to unmarshal JSON at line 1: "name" field is missing`,
	}, {
		name: "null name key",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":null,"value":"foo"}`,
		wantErr:       `read output file: failed to unmarshal JSON at line 1: "name" field value is null`,
	}, {
		name: "empty name key",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"","value":"foo"}`,
		wantErr:       `read output file: failed to unmarshal JSON at line 1: "name" field value is empty`,
	}, {
		name: "missing value key",
		outputs: map[string]*proto.Spec_Content_Output{
			"key": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"key"}`,
		wantErr:       `read output file: failed to unmarshal JSON at line 1: "value" field is missing`,
	}, {
		name: "null value key",
		outputs: map[string]*proto.Spec_Content_Output{
			"value": {Type: proto.ValueType_string},
		},
		writeToOutput: `{"name":"value","value":null}`,
		wantErr:       `read output file: failed to unmarshal JSON at line 1: "value" field value is null`,
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
