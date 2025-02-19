package runner

import (
	ctx "context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// SequenceOfSteps is a step that executes many steps.
type SequenceOfSteps struct {
	loadedFrom StepReference
	params     *Params
	specDef    *proto.SpecDefinition
	steps      []Step
}

func NewSequenceOfSteps(loadedFrom StepReference, params *Params, specDef *proto.SpecDefinition, steps ...Step) *SequenceOfSteps {
	return &SequenceOfSteps{
		loadedFrom: loadedFrom,
		params:     params,
		steps:      steps,
		specDef:    specDef,
	}
}

func (s *SequenceOfSteps) Describe() string {
	if len(s.steps) < 2 {
		return "sequence of steps"
	}

	return fmt.Sprintf("sequence of %d steps", len(s.steps))
}

func (s *SequenceOfSteps) Run(ctx ctx.Context, stepsCtx *StepsContext) (*proto.StepResult, error) {

	result := NewStepResultBuilder(s.loadedFrom, s.params, s.specDef)

	if err := result.ObserveEnv(stepsCtx.ExpandAndApplyEnv(s.specDef.Definition.Env)); err != nil {
		return result.BuildFailure(), fmt.Errorf("expand step env: %w", err)
	}

	for _, step := range s.steps {

		Breakpoint.At(s.specDef, stepsCtx)

		stepResult, err := result.ObserveStepResult(step.Run(ctx, stepsCtx))
		stepsCtx.RecordResult(stepResult)

		if err != nil {
			return result.BuildFailure(), err // expose underlying step error (no need to wrap)
		}
	}

	if err := result.ObserveOutputs(s.readOutputs(stepsCtx, stepsCtx.StepResults())); err != nil {
		return result.BuildFailure(), fmt.Errorf("outputs: %w", err)
	}

	return result.Build(), nil
}

func (s *SequenceOfSteps) readOutputs(stepsCtx *StepsContext, stepResults []*proto.StepResult) (map[string]*structpb.Value, error) {
	if s.specDef.Spec.Spec.OutputMethod == proto.OutputMethod_delegate {
		return findOutputsWithName(s.specDef.Definition.Delegate, stepResults)
	}

	return s.interpolateStepOutputs(stepsCtx)
}

func (s *SequenceOfSteps) interpolateStepOutputs(stepsCtx *StepsContext) (map[string]*structpb.Value, error) {
	outputs := make(map[string]*structpb.Value)

	for k, v := range s.specDef.Definition.Outputs {
		res, err := expression.Expand(stepsCtx.View(), v)
		if err == nil {
			outputs[k] = res.Value
		} else {
			return nil, fmt.Errorf("expand %q: %w", k, err)
		}
	}

	return outputs, nil
}

func findOutputsWithName(name string, results []*proto.StepResult) (map[string]*structpb.Value, error) {
	for _, s := range results {
		if s.Step != nil && s.Step.Name == name {
			return s.Outputs, nil
		}
	}

	return nil, fmt.Errorf("delegate: could not find step with name %q", name)
}
