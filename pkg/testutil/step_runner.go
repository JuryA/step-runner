package testutil

import (
	ctx "context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

type StepRunnerBuilder struct {
	t         *testing.T
	globalEnv map[string]string
	log       io.Writer
}

func StepRunner(t *testing.T) *StepRunnerBuilder {
	return &StepRunnerBuilder{
		t:         t,
		globalEnv: make(map[string]string),
		log:       os.Stdout,
	}
}

func (b *StepRunnerBuilder) WithLogs(log io.Writer) *StepRunnerBuilder {
	b.log = log
	return b
}

func (b *StepRunnerBuilder) WithGlobalCtxEnv(env map[string]string) *StepRunnerBuilder {
	b.globalEnv = env
	return b
}

func (b *StepRunnerBuilder) Run(yaml string) (*proto.StepResult, error) {
	schemaSpec, schemaStep, err := schema.ReadSteps(yaml)
	require.NoError(b.t, err)

	protoSpec, err := schemaSpec.Compile()
	require.NoError(b.t, err)

	protoDef, err := schemaStep.Compile()
	require.NoError(b.t, err)

	protoStepDef := &proto.SpecDefinition{Spec: protoSpec, Definition: protoDef}
	require.NoError(b.t, err)

	protoStepDef.Dir, err = os.Getwd()
	require.NoError(b.t, err)

	defs, err := cache.New()
	require.NoError(b.t, err)

	osEnv, err := runner.NewEnvironmentFromOS()
	require.NoError(b.t, err)

	globalCtx := runner.NewGlobalContext(osEnv)
	globalCtx.Env = runner.NewEnvironment(b.globalEnv)
	globalCtx.Stdout = b.log
	globalCtx.Stderr = b.log
	globalCtx.WorkDir, err = os.UserHomeDir()
	require.NoError(b.t, err)

	params := &runner.Params{}

	step, err := runner.NewParser(globalCtx, defs).Parse(protoStepDef, params, runner.StepDefinedInGitLabJob)
	require.NoError(b.t, err)

	inputs := params.NewInputsWithDefault(protoStepDef.Spec.Spec.Inputs)
	stepsCtx, err := runner.NewStepsContext(globalCtx, protoStepDef.Dir, inputs, globalCtx.Env)
	require.NoError(b.t, err)

	return step.Run(ctx.Background(), stepsCtx)
}
