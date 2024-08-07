package runner

import (
	"bytes"
	ctx "context"
	"maps"
	"os"
	"testing"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/schema/v1"

	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type runnerTest struct {
	name        string
	yaml        string
	globalEnv   map[string]string
	wantLog     string
	wantResults func(*testing.T, *proto.StepResult)
	wantErr     error
}

func requireStringEqualValue(t *testing.T, str string, got *structpb.Value) {
	want := structpb.NewStringValue(str)
	require.True(t, protobuf.Equal(want, got), "want %+v. got %+v", want, got)
}

func runTest(testCase runnerTest) func(*testing.T) {
	return func(t *testing.T) {
		stepDef, err := schema.ReadSteps(testCase.yaml, "")
		require.NoError(t, err)
		protoStepDef, err := schema.CompileSteps(stepDef)
		require.NoError(t, err)
		protoStepDef.Dir, _ = os.Getwd()

		defs, err := cache.New()
		require.NoError(t, err)
		runner, err := New(defs)
		require.NoError(t, err)

		var log bytes.Buffer

		globalCtx, err := NewGlobalContext()
		require.NoError(t, err)
		defer globalCtx.Cleanup()
		maps.Copy(globalCtx.Env, testCase.globalEnv)
		globalCtx.Stdout = &log
		globalCtx.Stderr = &log
		globalCtx.WorkDir, _ = os.UserHomeDir()

		params := &Params{}

		result, err := runner.Run(ctx.Background(), globalCtx, params, protoStepDef)
		if testCase.wantErr != nil {
			require.Error(t, err)
			require.Equal(t, testCase.wantErr.Error(), err.Error())
			if testCase.wantResults != nil {
				testCase.wantResults(t, result)
			}
		} else {
			require.NoError(t, err)
			if testCase.wantLog != "" {
				require.Equal(t, testCase.wantLog, log.String())
			}
			testCase.wantResults(t, result)
		}
	}
}
