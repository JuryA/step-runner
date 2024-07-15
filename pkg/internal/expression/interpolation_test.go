package expression

import (
	"errors"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"testing"

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
		value   *context.Variable
		want    *context.Variable
		wantErr error
	}{{
		value: context.NewStringVariable("", false),
		want:  context.NewStringVariable("", false),
	}, {
		value: context.NewStringVariable("${{job.job_id}}", false),
		want:  context.NewStringVariable("1982", false),
	}, {
		value: context.NewStringVariable("${{env.MOVIE}}", false),
		want:  context.NewStringVariable("tron", false),
	}, {
		value: context.NewStringVariable("${{env.WHERE}}", false),
		want:  context.NewStringVariable("inside", false),
	}, {
		value: context.NewStringVariable("${{inputs.name}}", false),
		want:  context.NewStringVariable("Kevin Flynn", false),
	}, {
		value: context.NewStringVariable("${{inputs.light_cycle}}", false),
		want: context.NewStructVariable(
			&structpb.Struct{Fields: map[string]*structpb.Value{
				"color":  structpb.NewStringValue("yellow"),
				"number": structpb.NewNumberValue(3),
			}},
			false),
	}, {
		value: context.NewStringVariable("PREFIX ${{inputs.light_cycle}} SUFFIX", false),
		want:  context.NewStringVariable(`PREFIX {"color":"yellow","number":3} SUFFIX`, false),
	}, {
		value: context.NewStringVariable("${{inputs.team_members}}", false),
		want: context.NewVariable(
			structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("tron"),
				structpb.NewStringValue("ram"),
				structpb.NewStringValue("flynn"),
			}}),
			false),
	}, {
		value: context.NewStructVariable(
			&structpb.Struct{Fields: map[string]*structpb.Value{
				"replace within": structpb.NewStringValue("${{inputs.name}}"),
			}},
			false),
		want: context.NewStructVariable(
			&structpb.Struct{Fields: map[string]*structpb.Value{
				"replace within": structpb.NewStringValue("Kevin Flynn"),
			}},
			false),
	}, {
		value: context.NewListVariable(
			&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("${{inputs.name}}"),
			}},
			false),
		want: context.NewListVariable(
			&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("Kevin Flynn"),
			}},
			false),
	}, {
		value: context.NewStringVariable("Hello, my name is ${{inputs.name}}. You killed my process. Prepare to SIGTERM.", false),
		want:  context.NewStringVariable("Hello, my name is Kevin Flynn. You killed my process. Prepare to SIGTERM.", false),
	}}
	for _, c := range cases {
		got, err := Expand(textContextSteps(), c.value)
		if c.wantErr != nil {
			require.Equal(t, c.wantErr, err)
		} else {
			require.Nil(t, err)
			require.Equal(t, c.want, got)
		}
	}
}
