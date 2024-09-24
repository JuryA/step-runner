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
	steps      []Step
}

func NewSequenceOfSteps(loadedFrom StepReference, params *Params, steps ...Step) *SequenceOfSteps {
	return &SequenceOfSteps{
		loadedFrom: loadedFrom,
		params:     params,
		steps:      steps,
	}
}

func (s *SequenceOfSteps) Describe() string {
	if len(s.steps) < 2 {
		return "sequence of steps"
	}

	return fmt.Sprintf("sequence of %d steps", len(s.steps))
}

func (s *SequenceOfSteps) Run(ctx ctx.Context, stepsCtx *StepsContext, specDefinition *proto.SpecDefinition) (*StepResult, error) {
	result := NewStepResultBuilder(s.loadedFrom, s.params, specDefinition)

	err := stepsCtx.ExpandAndApplyEnv(specDefinition.Definition.Env)
	result.WithEnv(stepsCtx.GetEnvs())

	if err != nil {
		return result.BuildFailure(), fmt.Errorf("failed to run %s: %w", s.Describe(), err)
	}

	// Create output and export files and add to context
	files, err := NewFiles(stepsCtx, specDefinition.Spec.Spec.OutputMethod, specDefinition.Spec.Spec.Outputs)

	if err != nil {
		return result.BuildFailure(), err
	}

	defer files.Cleanup()

	for _, step := range s.steps {
		stepResult, err := step.Run(ctx, stepsCtx, specDefinition)
		result.WithSubStepResult(stepResult)

		// Capture results even if there was an error
		if stepResult != nil {
			if hasStep, name := stepResult.StepName(); hasStep {
				stepsCtx.Steps[name] = stepResult.ProtoStepResult()
			}

			if stepResult.Failed() {
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
	if specDefinition.Spec.Spec.OutputMethod == proto.OutputMethod_delegate {
		outputs, err := result.BuildSubStepResults().FindOutputsForStepName(specDefinition.Definition.Delegate)
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

	for k, v := range specDefinition.Definition.Outputs {
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
