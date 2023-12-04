package expression

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestEvaluate(t *testing.T) {
	cases := []struct {
		value   string
		want    *structpb.Value
		wantErr error
	}{{
		value: "job.job_id",
		want:  structpb.NewStringValue("1982"),
	}, {
		value: "  job.job_id  ",
		want:  structpb.NewStringValue("1982"),
	}, {
		value:   "job.undefined_key",
		wantErr: errors.New(`"job.undefined_key" cannot be evaluated`),
	}}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			got, err := Evaluate(textContextSteps(), c.value)
			if c.wantErr != nil {
				require.Equal(t, c.wantErr, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, c.want, got)
			}
		})
	}
}
