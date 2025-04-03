package testutil

import (
	"bytes"
	ctx "context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

type StepRunnerBuilder struct {
	t            *testing.T
	globalEnv    map[string]string
	globalCtxJob map[string]string
	log          *bytes.Buffer
}

func StepRunner(t *testing.T) *StepRunnerBuilder {
	return &StepRunnerBuilder{
		t:            t,
		globalEnv:    make(map[string]string),
		globalCtxJob: make(map[string]string),
		log:          &bytes.Buffer{},
	}
}

func (b *StepRunnerBuilder) WithGlobalCtxEnv(env map[string]string) *StepRunnerBuilder {
	b.globalEnv = env
	return b
}

func (b *StepRunnerBuilder) WithEnvKeyVal(key, value string) *StepRunnerBuilder {
	b.globalEnv[key] = value
	return b
}

func (b *StepRunnerBuilder) WithGlobalCtxJob(key, value string) *StepRunnerBuilder {
	b.globalCtxJob[key] = value
	return b
}

func (b *StepRunnerBuilder) WithDebugLogs() *StepRunnerBuilder {
	b.globalCtxJob[runner.LogLevelEnvName] = "debug"
	return b
}

func (b *StepRunnerBuilder) Run(yaml string) (*proto.StepResult, string, error) {
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

	env, err := runner.GlobalEnvironment(osEnv, b.globalCtxJob)
	require.NoError(b.t, err)

	if b.globalEnv != nil {
		env = env.AddLexicalScope(b.globalEnv)
	}

	workDir, err := os.UserHomeDir()
	require.NoError(b.t, err)

	globalCtx := runner.NewGlobalContext(workDir, b.globalCtxJob, env, b.log, b.log)
	params := &runner.Params{}

	step, err := runner.NewParser(defs).Parse(globalCtx, protoStepDef, params, runner.StepDefinedInGitLabJob)
	require.NoError(b.t, err)

	inputs := params.NewInputsWithDefault(protoStepDef.Spec.Spec.Inputs)
	stepsCtx, err := runner.NewStepsContext(globalCtx, protoStepDef.Dir, inputs, globalCtx.EnvWithLexicalScope(params.Env))
	require.NoError(b.t, err)

	run, err := step.Run(ctx.Background(), stepsCtx)

	b.t.Cleanup(func() { fmt.Println(b.log) })
	return run, b.log.String(), err
}
