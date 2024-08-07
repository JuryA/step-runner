package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestStepsContext_ExpandAndApplyEnv(t *testing.T) {
	globalCtx, err := NewGlobalContext()
	require.NoError(t, err)

	inputs := map[string]*structpb.Value{"name": structpb.NewStringValue("sally")}
	env := map[string]string{"HOME": "/home"}

	stepsCtx := NewStepsContext(globalCtx, "", inputs, env)
	err = stepsCtx.ExpandAndApplyEnv(map[string]string{"WORK_DIR": "/home/${{ inputs.name }}"})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"HOME": "/home", "WORK_DIR": "/home/sally"}, stepsCtx.Env)
}
