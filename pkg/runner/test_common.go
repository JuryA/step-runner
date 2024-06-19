package runner

import (
	"bytes"
	ctx "context"
	"maps"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"gitlab.com/gitlab-org/step-runner/proto"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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

func testCases(t *testing.T, cases []runnerTest) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stepDef, err := step.ReadSteps(c.yaml, "")
			require.NoError(t, err)
			protoStepDef, err := step.CompileSteps(stepDef)
			require.NoError(t, err)
			protoStepDef.Dir, _ = os.Getwd()

			defs, err := cache.New()
			require.NoError(t, err)
			runner, err := New(defs, nil)
			require.NoError(t, err)

			var log bytes.Buffer

			globalCtx, err := context.NewGlobal()
			require.NoError(t, err)
			defer globalCtx.Cleanup()
			maps.Copy(globalCtx.Env, c.globalEnv)
			globalCtx.Stdout = &log
			globalCtx.Stderr = &log
			globalCtx.WorkDir, _ = os.UserHomeDir()

			params := &Params{}

			result, err := runner.Run(ctx.Background(), globalCtx, params, protoStepDef)
			if c.wantErr != nil {
				require.Equal(t, c.wantErr.Error(), err.Error())
				if c.wantResults != nil {
					c.wantResults(t, result)
				}
			} else {
				require.NoError(t, err)
				if c.wantLog != "" {
					require.Equal(t, c.wantLog, log.String())
				}
				c.wantResults(t, result)
			}
		})
	}
}
