package expression_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func textContextSteps(t *testing.T) *expression.InterpolationContext {
	stepsCtx := bldr.StepsContext(t).
		WithGlobalContext(bldr.GlobalContext().WithJob("job_id", "1982").Build()).
		WithEnv("MOVIE", "tron").
		WithEnv("WHERE", "inside").
		WithInput("name", structpb.NewStringValue("Kevin Flynn")).
		WithInput("light_cycle", structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
			"color":  structpb.NewStringValue("yellow"),
			"number": structpb.NewNumberValue(3),
		}})).
		WithInput("team_members", structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
			structpb.NewStringValue("tron"),
			structpb.NewStringValue("ram"),
			structpb.NewStringValue("flynn"),
		}})).
		Build()

	return stepsCtx.View()
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
			got, err := expression.ExpandString(textContextSteps(t), c.value)
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
		got, err := expression.Expand(textContextSteps(t), c.value)
		if c.wantErr != nil {
			require.Equal(t, c.wantErr, err)
		} else {
			require.Nil(t, err)
			require.Equal(t, c.want, got.Value)
			require.False(t, got.Sensitive)
		}
	}
}
