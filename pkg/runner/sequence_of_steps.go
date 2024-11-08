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

func (s *SequenceOfSteps) Run(ctx ctx.Context, stepsCtx *StepsContext, globalCtx *GlobalContext, stepDir string, inputs map[string]*structpb.Value, env *Environment, steps map[string]*proto.StepResult) (*proto.StepResult, error) {
	stepsCtx, err := NewStepsContext(globalCtx, stepDir, inputs, env, steps)
	if err != nil {
		return nil, err
	}

	defer stepsCtx.Cleanup()

	result := NewStepResultBuilder(s.loadedFrom, s.params, s.specDef)

	err = stepsCtx.ExpandAndApplyEnv(s.specDef.Definition.Env)
	result.WithEnv(stepsCtx.GetEnvs())

	if err != nil {
		return result.BuildFailure(), fmt.Errorf("failed to run %s: %w", s.Describe(), err)
	}

	for _, step := range s.steps {
		stepResult, err := step.Run(ctx, stepsCtx, globalCtx, stepDir, inputs, stepsCtx.Env, stepsCtx.Steps)
		result.WithSubStepResult(stepResult)

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
		outputs, err := findOutputsWithName(s.specDef.Definition.Delegate, result.subStepResults)
		result.WithMergedOutputs(outputs)

		if err != nil {
			return result.BuildFailure(), err
		}

		return result.Build(), nil
	}

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
			fmt.Fprintf(stepsCtx.GlobalContext.Stderr, "Cannot assign %q due to error: %s", k, resErr.Error())
		}
	}

	result.WithMergedOutputs(expandedOutputs)
	return result.Build(), nil
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
