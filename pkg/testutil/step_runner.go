package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"gitlab.com/gitlab-org/step-runner/pkg/di"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

type StepRunnerBuilder struct {
	t            *testing.T
	globalEnv    map[string]string
	globalCtxJob map[string]string
	log          *bytes.Buffer
	timeout      time.Duration
}

func StepRunner(t *testing.T) *StepRunnerBuilder {
	return &StepRunnerBuilder{
		t:            t,
		globalEnv:    make(map[string]string),
		globalCtxJob: make(map[string]string),
		log:          &bytes.Buffer{},
		timeout:      20 * time.Second,
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

	dir, err := os.Getwd()
	require.NoError(b.t, err)

	specDef := runner.NewSpecDefinition(protoSpec, protoDef, dir)
	require.NoError(b.t, err)

	diContainer := di.NewContainer()

	osEnv, err := runner.NewEnvironmentFromOS()
	require.NoError(b.t, err)

	env, err := runner.GlobalEnvironment(osEnv, b.globalCtxJob)
	require.NoError(b.t, err)

	if b.globalEnv != nil {
		env = env.AddLexicalScope(b.globalEnv)
	}

	workDir, err := os.UserHomeDir()
	require.NoError(b.t, err)

	stepLog := io.MultiWriter(b.log, os.Stdout)
	globalCtx := runner.NewGlobalContext(workDir, b.globalCtxJob, env, stepLog, stepLog)
	params := &runner.Params{}

	stepParser, err := diContainer.StepParser()
	require.NoError(b.t, err)

	step, err := stepParser.Parse(globalCtx, specDef, params, runner.StepDefinedInGitLabJob)
	require.NoError(b.t, err)

	inputs := params.NewInputsWithDefault(specDef.SpecInputs())
	stepsCtx, err := runner.NewStepsContext(globalCtx, specDef.Dir(), inputs, globalCtx.EnvWithLexicalScope(params.Env))
	require.NoError(b.t, err)

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	run, err := step.Run(ctx, stepsCtx)

	return run, b.log.String(), err
}
