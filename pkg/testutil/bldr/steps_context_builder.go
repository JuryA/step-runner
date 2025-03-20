package bldr

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepsContextBuilder struct {
	t           *testing.T
	globalCtx   *runner.GlobalContext
	env         map[string]string
	inputs      map[string]*structpb.Value
	outputFile  *runner.StepFile
	exportFile  *runner.StepFile
	stepResults map[string]*proto.StepResult
}

func StepsContext(t *testing.T) *StepsContextBuilder {
	outputFile, err := runner.NewStepFileInDir(t.TempDir())
	require.NoError(t, err)

	exportFile, err := runner.NewStepFileInDir(t.TempDir())
	require.NoError(t, err)

	return &StepsContextBuilder{
		t:           t,
		globalCtx:   GlobalContext().Build(),
		env:         map[string]string{},
		inputs:      map[string]*structpb.Value{},
		outputFile:  outputFile,
		exportFile:  exportFile,
		stepResults: map[string]*proto.StepResult{},
	}
}

func (bldr *StepsContextBuilder) WithGlobalContext(globalCtx *runner.GlobalContext) *StepsContextBuilder {
	bldr.globalCtx = globalCtx
	return bldr
}

func (bldr *StepsContextBuilder) WithEnv(key, value string) *StepsContextBuilder {
	bldr.env[key] = value
	return bldr
}

func (bldr *StepsContextBuilder) WithInput(name string, value *structpb.Value) *StepsContextBuilder {
	bldr.inputs[name] = value
	return bldr
}

func (bldr *StepsContextBuilder) WithStepResults(stepResults map[string]*proto.StepResult) *StepsContextBuilder {
	bldr.stepResults = stepResults
	return bldr
}

func (bldr *StepsContextBuilder) Build() *runner.StepsContext {
	stepsCtx, err := runner.NewStepsContext(
		bldr.globalCtx,
		".",
		bldr.inputs,
		runner.NewEnvironment(bldr.env),
		runner.WithStepsCtxOutputFile(bldr.outputFile),
		runner.WithStepsCtxExportFile(bldr.exportFile),
		runner.WithStepsCtxStepResults(bldr.stepResults))
	require.NoError(bldr.t, err)

	return stepsCtx
}
