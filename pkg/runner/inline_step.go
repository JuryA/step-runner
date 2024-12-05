package runner

import (
	ctx "context"
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type InlineStepFn func(context.Context, *StepsContext) error

// InlineStep is a step that executes a known function as a step.
type InlineStep struct {
	loadedFrom   StepReference
	params       *Params
	specDef      *proto.SpecDefinition
	inlineStepFn InlineStepFn
}

func NewInlineStep(loadedFrom StepReference, inlineStepFn InlineStepFn, params *Params, specDef *proto.SpecDefinition) *InlineStep {
	return &InlineStep{
		loadedFrom:   loadedFrom,
		inlineStepFn: inlineStepFn,
		params:       params,
		specDef:      specDef,
	}
}

func (s *InlineStep) Describe() string {
	return fmt.Sprintf("step %q", strings.Join(s.specDef.Definition.Exec.Command, " "))
}

func (s *InlineStep) Run(ctx ctx.Context, stepsCtx *StepsContext) (*proto.StepResult, error) {
	if err := stepsCtx.Logln("Running step %q", s.loadedFrom.Describe()); err != nil {
		return nil, err
	}

	result := NewStepResultBuilder(s.loadedFrom, s.params, s.specDef)

	if err := result.ObserveEnv(stepsCtx.ExpandAndApplyEnv(s.specDef.Definition.Env)); err != nil {
		return result.BuildFailure(), fmt.Errorf("expand step env: %w", err)
	}

	if err := s.inlineStepFn(ctx, stepsCtx); err != nil {
		return result.BuildFailure(), fmt.Errorf("execFn: %w", err)
	}

	exports, err := result.ObserveExports(stepsCtx.ExportFile.ReadEnvironment())
	if err != nil {
		return result.BuildFailure(), fmt.Errorf("export file: %w", err)
	}

	stepsCtx.GlobalContext.Env.Mutate(exports)

	stepResult, err := s.readOutputs(stepsCtx.OutputFile)
	if err != nil {
		return result.BuildFailure(), fmt.Errorf("read outputs: %w", err)
	}

	return stepResult, nil
}

func (s *InlineStep) readOutputs(outputFile *StepFile) (*proto.StepResult, error) {
	if s.specDef.Spec.Spec.OutputMethod != proto.OutputMethod_delegate {
		return nil, fmt.Errorf("expected inline step to delegate outputs")
	}

	stepResult, err := outputFile.ReadStepResult()
	if err != nil {
		return nil, fmt.Errorf("delegate: %w", err)
	}

	return stepResult, nil
}
