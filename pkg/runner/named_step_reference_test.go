package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestNamedStepReference_ToProtoStep(t *testing.T) {
	t.Run("converts inputs and environment", func(t *testing.T) {
		params := &Params{
			Inputs: map[string]*context.Variable{"greeting": context.NewVariable(structpb.NewStringValue("hello"), false)},
			Env:    map[string]string{"VALUE": "value"},
		}

		stepRef := NewNamedStepReference("step.name", &proto.Step_Reference{})
		protoStep := stepRef.ToProtoStep(params)

		require.Equal(t, map[string]*structpb.Value{"greeting": structpb.NewStringValue("hello")}, protoStep.Inputs)
		require.Equal(t, map[string]string{"VALUE": "value"}, protoStep.Env)
	})

	t.Run("converts to proto Step", func(t *testing.T) {
		ref := &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "git://gitlab.com/repository",
			Filename: "step.yml",
			Version:  "@1",
		}

		stepRef := NewNamedStepReference("step.name", ref)
		protoStep := stepRef.ToProtoStep(&Params{})

		require.Equal(t, "step.name", protoStep.Name)
		require.Equal(t, ref, protoStep.Step)
	})

	t.Run("converts to proto Step when no step reference", func(t *testing.T) {
		stepRef := NewNamedStepReference("", nil)
		protoStep := stepRef.ToProtoStep(&Params{})

		require.Equal(t, "", protoStep.Name)
		require.Nil(t, protoStep.Step)
	})
}
