package runner

import (
	ctx "context"
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type RunStepFn func(ctx context.Context, stepsCtx *StepsContext) error

// InMemoryStep is a step that executes a command by calling a Go function.
type InMemoryStep struct {
	name   string
	spec   *proto.Spec
	stepFn RunStepFn
}

func NewInMemoryStep(name string, spec *proto.Spec, stepFn RunStepFn) *InMemoryStep {
	return &InMemoryStep{
		name:   name,
		spec:   spec,
		stepFn: stepFn,
	}
}

func (s *InMemoryStep) Describe() string {
	return fmt.Sprintf("in-memory step %q", s.name)
}

func (s *InMemoryStep) Run(ctx ctx.Context, stepsCtx *StepsContext) (*proto.StepResult, error) {
	if err := stepsCtx.Logln("Running in memory step %q", s.name); err != nil {
		return nil, err
	}

	result := NewStepResultBuilder(nil, nil, nil)

	if err := result.ObserveEnv(func() (*Environment, error) { return stepsCtx.Env, nil }()); err != nil {
		return result.BuildFailure(), fmt.Errorf("expand step env: %w", err)
	}

	if err := result.ObserveOutputs(s.readOutputs(stepsCtx.OutputFile)); err != nil {
		return result.BuildFailure(), fmt.Errorf("output file: %w", err)
	}

	if err := result.ObserveExecutedCmd(s.execute(ctx, stepsCtx)); err != nil {
		return result.BuildFailure(), fmt.Errorf("exec: %w", err)
	}

	exports, err := result.ObserveExports(stepsCtx.ExportFile.ReadEnvironment())
	if err != nil {
		return result.BuildFailure(), fmt.Errorf("export file: %w", err)
	}

	stepsCtx.GlobalContext.Env.Mutate(exports)
	return result.Build(), nil
}

func (s *InMemoryStep) execute(ctx ctx.Context, stepsCtx *StepsContext) (*ExecResult, error) {
	funcArgs := []string{}

	err := s.stepFn(ctx, stepsCtx)
	exitCode := 0

	if err != nil {
		exitCode = 1
	}

	execResult := NewExecResult(".", funcArgs, exitCode)
	return execResult, err
}

func (s *InMemoryStep) readOutputs(outputFile *StepFile) (map[string]*structpb.Value, error) {
	if s.spec.Spec.OutputMethod == proto.OutputMethod_delegate {
		stepResult, err := outputFile.ReadStepResult()
		if err != nil {
			return nil, fmt.Errorf("delegate: %w", err)
		}

		return stepResult.Outputs, nil
	}

	return outputFile.ReadValues(s.spec.Spec.Outputs)
}
