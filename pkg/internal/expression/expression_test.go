package expression_test

import (
	"errors"
	"testing"

	"gitlab.com/gitlab-org/step-runner/pkg/context/b"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"

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
		wantErr: errors.New(`job.undefined_key: the "undefined_key" was not found`),
	}}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			got, err := expression.Evaluate(textContextSteps(), c.value)
			if c.wantErr != nil {
				require.Equal(t, c.wantErr, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, c.want, got.Value)
			}
		})
	}
}

func TestEvaluateSensitivity(t *testing.T) {
	tests := map[string]struct {
		sensitive           bool
		wantSensitiveReason string
	}{
		"sensitive": {
			sensitive:           true,
			wantSensitiveReason: "steps.secret_factory.outputs.secret",
		},
		"not sensitive": {
			sensitive: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stepResult := b.ProtoStepResult().
				WithName("secret_factory").
				WithOutputSpec("secret", &proto.Spec_Content_Output{Type: proto.ValueType_string, Sensitive: test.sensitive}).
				WithOutput("secret", structpb.NewStringValue("secret.value")).
				Build()
			stepContext := b.StepContext().WithStepResult(stepResult).Build()

			value, err := expression.Evaluate(stepContext, "steps.secret_factory.outputs.secret")
			require.NoError(t, err)
			require.Equal(t, structpb.NewStringValue("secret.value"), value.Value)
			require.Equal(t, test.sensitive, value.Sensitive)

			if test.sensitive {
				require.Equal(t, test.wantSensitiveReason, value.SensitiveReason)
			}
		})
	}
}
