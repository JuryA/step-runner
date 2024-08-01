package expression

import (
	"errors"
	"testing"

	"gitlab.com/gitlab-org/step-runner/proto"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type GlobalContext struct {
	Job map[string]string `json:"job"`
}

type StepContext struct {
	*GlobalContext
	Env    map[string]string          `json:"env"`
	Inputs map[string]*structpb.Value `json:"inputs"`
}

func textContextSteps() *StepContext {
	return &StepContext{
		GlobalContext: &GlobalContext{
			Job: map[string]string{
				"job_id": "1982",
			},
		},
		Env: map[string]string{
			"MOVIE": "tron",
			"WHERE": "inside",
		},
		Inputs: map[string]*structpb.Value{
			"name": structpb.NewStringValue("Kevin Flynn"),
			"light_cycle": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
				"color":  structpb.NewStringValue("yellow"),
				"number": structpb.NewNumberValue(3),
			}}),
			"team_members": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("tron"),
				structpb.NewStringValue("ram"),
				structpb.NewStringValue("flynn"),
			}}),
		},
	}
}

func TestExpandString(t *testing.T) {
	cases := []struct {
		value   string
		want    string
		wantErr error
	}{{
		value: "${{job.job_id}}",
		want:  "1982",
	}, {
		value: "${{ job.job_id }}",
		want:  "1982",
	}, {
		value: "${{ job.job_id }}:${{ env.MOVIE }}",
		want:  "1982:tron",
	}, {
		value:   "${{ job.undefined_key }}",
		wantErr: errors.New(`job.undefined_key: the "undefined_key" was not found`),
	}, {
		value:   `${{ job["${{ job.key }}"] }}`,
		wantErr: errors.New(`job["${{ job.key }}"]: the "job[\"${{ job" was not found`),
	}, {
		value:   `${{ job.job_id`,
		wantErr: errors.New(`The " job.job_id" is not closed: ${{ ... }}`),
	}, {
		value:   `${{ job.job_id }} }}`,
		wantErr: errors.New(`The " job.job_id }} " has extra '}}'`),
	}, {
		value: `${{inputs.light_cycle}}`,
		want:  `{"color":"yellow","number":3}`,
	}, {
		value: `PREFIX ${{inputs.light_cycle}} SUFFIX`,
		want:  `PREFIX {"color":"yellow","number":3} SUFFIX`,
	}}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			got, err := ExpandString(textContextSteps(), c.value)
			if c.wantErr != nil {
				require.Equal(t, c.wantErr, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, c.want, got)
			}
		})
	}
}

func TestExpand(t *testing.T) {
	cases := []struct {
		value   *structpb.Value
		want    *structpb.Value
		wantErr error
	}{{
		value: structpb.NewStringValue(""),
		want:  structpb.NewStringValue(""),
	}, {
		value: structpb.NewStringValue("${{job.job_id}}"),
		want:  structpb.NewStringValue("1982"),
	}, {
		value: structpb.NewStringValue("${{env.MOVIE}}"),
		want:  structpb.NewStringValue("tron"),
	}, {
		value: structpb.NewStringValue("${{env.WHERE}}"),
		want:  structpb.NewStringValue("inside"),
	}, {
		value: structpb.NewStringValue("${{inputs.name}}"),
		want:  structpb.NewStringValue("Kevin Flynn"),
	}, {
		value: structpb.NewStringValue("${{inputs.light_cycle}}"),
		want: structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
			"color":  structpb.NewStringValue("yellow"),
			"number": structpb.NewNumberValue(3),
		}}),
	}, {
		value: structpb.NewStringValue("PREFIX ${{inputs.light_cycle}} SUFFIX"),
		want:  structpb.NewStringValue(`PREFIX {"color":"yellow","number":3} SUFFIX`),
	}, {
		value: structpb.NewStringValue("${{inputs.team_members}}"),
		want: structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
			structpb.NewStringValue("tron"),
			structpb.NewStringValue("ram"),
			structpb.NewStringValue("flynn"),
		}}),
	}, {
		value: structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
			"replace within": structpb.NewStringValue("${{inputs.name}}"),
		}}),
		want: structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
			"replace within": structpb.NewStringValue("Kevin Flynn"),
		}}),
	}, {
		value: structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
			structpb.NewStringValue("${{inputs.name}}"),
		}}),
		want: structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
			structpb.NewStringValue("Kevin Flynn"),
		}}),
	}, {
		value: structpb.NewStringValue("Hello, my name is ${{inputs.name}}. You killed my process. Prepare to SIGTERM."),
		want:  structpb.NewStringValue("Hello, my name is Kevin Flynn. You killed my process. Prepare to SIGTERM."),
	}}
	for _, c := range cases {
		got, err := Expand(textContextSteps(), c.value)
		if c.wantErr != nil {
			require.Equal(t, c.wantErr, err)
		} else {
			require.Nil(t, err)
			require.Equal(t, c.want, got.Value)
			require.False(t, got.Sensitive)
		}
	}
}

func TestExpandSensitivity(t *testing.T) {
	tests := map[string]struct {
		stepResult          *proto.StepResult
		template            *structpb.Value
		wantValue           *structpb.Value
		wantSensitive       bool
		wantSensitiveReason string
	}{
		"contains a sensitive value": {
			stepResult: b.protoStepResult().
				withName("secret_factory").
				withOutputSpec("secret", &proto.Spec_Content_Output{Type: proto.ValueType_string, Sensitive: true}).
				withOutputSpec("engine", &proto.Spec_Content_Output{Type: proto.ValueType_string, Sensitive: false}).
				withOutput("secret", structpb.NewStringValue("secret.value")).
				withOutput("engine", structpb.NewStringValue("hard-coded")).
				build(),
			template:            structpb.NewStringValue("a secret factory using the ${{ steps.secret_factory.outputs.engine }} engine generated ${{ steps.secret_factory.outputs.secret }}"),
			wantValue:           structpb.NewStringValue("a secret factory using the hard-coded engine generated secret.value"),
			wantSensitive:       true,
			wantSensitiveReason: "steps.secret_factory.outputs.secret",
		},
		"contains no sensitive values": {
			stepResult: b.protoStepResult().
				withName("word-of-the-day").
				withOutputSpec("word", &proto.Spec_Content_Output{Type: proto.ValueType_string, Sensitive: false}).
				withOutput("word", structpb.NewStringValue("collywobbles")).
				build(),
			template:      structpb.NewStringValue("word of the day is ${{ steps.word-of-the-day.outputs.word }}"),
			wantValue:     structpb.NewStringValue("word of the day is collywobbles"),
			wantSensitive: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stepContext := b.stepContext().withStepResult(test.stepResult).build()

			value, err := Expand(stepContext, test.template)
			require.NoError(t, err)
			require.Equal(t, test.wantValue, value.Value)
			require.Equal(t, test.wantSensitive, value.Sensitive)

			if test.wantSensitive {
				require.Equal(t, test.wantSensitiveReason, value.SensitiveReason)
			}
		})
	}
}
