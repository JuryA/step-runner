package expression_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/api/expression"
)

func TestExpand(t *testing.T) {
	jobInputs := []*expression.StepsJobInput{
		{
			Key: "light_cycle",
			Value: structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"color":  structpb.NewStringValue("yellow"),
					"number": structpb.NewNumberValue(3),
				}}),
			Sensitive: false,
		},
	}

	expanded, err := expression.Expand(jobInputs, structpb.NewStringValue("${{inputs.light_cycle.color}}"))
	require.Nil(t, err)
	require.Equal(t, "yellow", expanded.GetStringValue())
}
