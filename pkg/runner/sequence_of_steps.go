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
	steps []Step
}

func NewSequenceOfSteps(steps ...Step) *SequenceOfSteps {
	return &SequenceOfSteps{
		steps: steps,
	}
}

func (s *SequenceOfSteps) Describe() string {
	if len(s.steps) < 2 {
		return "sequence of steps"
	}

	return fmt.Sprintf("sequence of %d steps", len(s.steps))
}

func (s *SequenceOfSteps) Run(ctx ctx.Context, stepsCtx *StepsContext, specDefinition *proto.SpecDefinition) (*proto.StepResult, error) {
	result := &proto.StepResult{
		SpecDefinition: specDefinition,
		Status:         proto.StepResult_failure,
		Outputs:        make(map[string]*structpb.Value),
		Exports:        make(map[string]string),
	}

	err := stepsCtx.ExpandAndApplyEnv(specDefinition.Definition.Env)

	if err != nil {
		return result, fmt.Errorf("failed to run %s: %w", s.Describe(), err)
	}

	result.Env = stepsCtx.GetEnvs()

	// Create output and export files and add to context
	files, err := NewFiles(stepsCtx, specDefinition.Spec.Spec.OutputMethod, specDefinition.Spec.Spec.Outputs)

	if err != nil {
		return result, err
	}

	defer files.Cleanup()

	for _, step := range s.steps {
		stepResult, err := step.Run(ctx, stepsCtx, specDefinition)

		// Capture results even if there was an error
		if stepResult != nil {
			result.SubStepResults = append(result.SubStepResults, stepResult)

			if stepResult.Step != nil {
				stepsCtx.Steps[stepResult.Step.Name] = stepResult
			}

			if stepResult.Status == proto.StepResult_failure {
				return result, fmt.Errorf("failed to run %s: %w", s.Describe(), err)
			}
		}

		if err != nil {
			return result, err
		}
	}

	// Delegate outputs are surfaced directly, effectively making
	// the delegation mechanism "disappear" from the execution
	// context.
	if specDefinition.Spec.Spec.OutputMethod == proto.OutputMethod_delegate {
		err := mergeDelegateOutput(specDefinition.Definition.Delegate, result)

		if err != nil {
			return result, err
		}

		result.Status = proto.StepResult_success
		return result, nil
	}

	// Expand step definition outputs which may reference outputs
	// of sub-steps. Outputs of sub-steps will not be available
	// for reference after returning, which would break
	// encapsulation of the step function.
	for k, v := range specDefinition.Definition.Outputs {
		res, resErr := expression.Expand(stepsCtx, v)
		if resErr == nil {
			result.Outputs[k] = res.Value
		} else {
			fmt.Fprintf(stepsCtx.GlobalContext.Stderr, "Cannot assign %q due to error: %s", k, resErr.Error())
		}
	}

	result.Status = proto.StepResult_success
	return result, nil
}

// mergeDelegateOutput copies outputs from the designated delegate sub-step.
func mergeDelegateOutput(
	delegate string,
	result *proto.StepResult,
) error {
	for _, s := range result.SubStepResults {
		if s.Step != nil && s.Step.Name == delegate {
			for k, v := range s.Outputs {
				result.Outputs[k] = v
			}
			return nil
		}
	}
	return fmt.Errorf("delegating outputs to %q: could not find substep", delegate)
}
