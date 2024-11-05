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
	result := NewStepResultBuilder(s.loadedFrom, s.params, s.specDef, stepsCtx)

	if err := stepsCtx.ExpandAndApplyEnv(s.specDef.Definition.Env); err != nil {
		return result.BuildFailure(), fmt.Errorf("failed to run %s: %w", s.Describe(), err)
	}

	// Create output and export files and add to context
	files, err := NewFiles(stepsCtx, s.specDef.Spec.Spec.OutputMethod, s.specDef.Spec.Spec.Outputs)

	if err != nil {
		return result.BuildFailure(), err
	}

	defer files.Cleanup()

	for _, step := range s.steps {
		stepResult, err := result.ObserveSubStepResult(step.Run(ctx, stepsCtx))

		// Capture results even if there was an error
		if stepResult != nil {
			if stepResult.Step != nil {
				stepsCtx.Steps[stepResult.Step.Name] = stepResult
			}

			if stepResult.Status == proto.StepResult_failure {
				return result.BuildFailure(), fmt.Errorf("failed to run %s: %w", s.Describe(), err)
			}
		}

		if err != nil {
			return result.BuildFailure(), err
		}
	}

	// Delegate outputs are surfaced directly, effectively making
	// the delegation mechanism "disappear" from the execution
	// context.
	if s.specDef.Spec.Spec.OutputMethod == proto.OutputMethod_delegate {
		if err := result.ObserveMergedOutputs(findOutputsWithName(s.specDef.Definition.Delegate, result.subStepResults)); err != nil {
			return result.BuildFailure(), err
		}

		return result.Build(), nil
	}

	if err := result.ObserveMergedOutputs(s.expandOutputs(stepsCtx)); err != nil {
		return result.BuildFailure(), err
	}

	return result.Build(), nil
}

func (s *SequenceOfSteps) expandOutputs(stepsCtx *StepsContext) (map[string]*structpb.Value, error) {
	// Expand step definition outputs which may reference outputs
	// of sub-steps. Outputs of sub-steps will not be available
	// for reference after returning, which would break
	// encapsulation of the step function.
	expandedOutputs := make(map[string]*structpb.Value)

	for k, v := range s.specDef.Definition.Outputs {
		res, resErr := expression.Expand(stepsCtx.View(), v)
		if resErr == nil {
			expandedOutputs[k] = res.Value
		} else {
			return nil, fmt.Errorf("cannot assign %q due to error: %s", k, resErr.Error())
		}
	}

	return expandedOutputs, nil
}

// findOutputsWithName finds the output results for the step by step name
func findOutputsWithName(name string, results []*proto.StepResult) (map[string]*structpb.Value, error) {
	for _, s := range results {
		if s.Step != nil && s.Step.Name == name {
			return s.Outputs, nil
		}
	}

	return nil, fmt.Errorf("delegating outputs to %q: could not find substep", name)
}
