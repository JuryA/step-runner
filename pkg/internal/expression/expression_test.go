package expression

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
)

func TestInterpolateProtoValue(t *testing.T) {
	stepsCtx := &context.Steps{
		Global: &context.Global{
			Job: map[string]string{
				"job_id": "1982",
			},
			Env: map[string]string{
				"MOVIE": "tron",
			},
		},
		Env: map[string]string{
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
		Outputs: map[string]map[string]string{},
	}
	cases := []struct {
		value *structpb.Value
		want  *structpb.Value
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
		want:  structpb.NewStringValue(`{"color":"yellow","number":3}`),
	}, {
		value: structpb.NewStringValue("${{inputs.team_members}}"),
		want:  structpb.NewStringValue(`["tron","ram","flynn"]`),
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
		got := InterpolateProtoValue(stepsCtx, c.value)
		require.Equal(t, c.want, got)
	}
}
