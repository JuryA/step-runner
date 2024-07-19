package ci

import (
	"os"
	"testing"

	"gitlab.com/gitlab-org/step-runner/proto"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCICmd(t *testing.T) {
	steps := `
- name: secret_factory_a
  step: ../../pkg/runner/test_steps/secret_factory
- name: secret_factory_b
  step: ../../pkg/runner/test_steps/secret_factory
  inputs:
    secret_override: ${{ steps.secret_factory_a.outputs.secret }}
`

	require.NoError(t, os.Setenv("STEPS", steps))
	defer func() { _ = os.Unsetenv("STEPS") }()

	err := run(nil, nil)
	require.NoError(t, err)

	file, err := os.ReadFile("step-results.json")
	require.NoError(t, err)

	var msg proto.StepResult
	err = protojson.Unmarshal(file, &msg)
	require.NoError(t, err)
	require.Equal(t, proto.StepResult_success, msg.Status)

	require.Len(t, msg.SubStepResults, 2)
	require.Equal(t, msg.SubStepResults[0].Outputs["secret"], msg.SubStepResults[1].Outputs["secret"])
}
