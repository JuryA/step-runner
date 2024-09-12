package runner_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestStepsContext_ExpandAndApplyEnv(t *testing.T) {
	globalCtx := bldr.GlobalContext().Build()

	inputs := map[string]*structpb.Value{"name": structpb.NewStringValue("sally")}
	env := map[string]string{"HOME": "/home"}

	stepsCtx := runner.NewStepsContext(globalCtx, "", inputs, env)
	err := stepsCtx.ExpandAndApplyEnv(map[string]string{"WORK_DIR": "/home/${{ inputs.name }}"})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"HOME": "/home", "WORK_DIR": "/home/sally"}, stepsCtx.Env)
}
