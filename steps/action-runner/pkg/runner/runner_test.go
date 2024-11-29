package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/proto"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRun(t *testing.T) {
	got, err := Run("./test_actions/echo", "catthehacker/ubuntu:act-latest", map[string]string{
		"message": "hello world",
	})
	require.NoError(t, err)
	want := &proto.StepResult{
		Outputs: map[string]*structpb.Value{
			"echo": structpb.NewStringValue("hello world"),
		},
	}
	require.True(t, protobuf.Equal(want, got), "want %+v. got %+v", want, got)
}
