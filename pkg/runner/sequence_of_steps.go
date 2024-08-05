package runner

import (
	ctx "context"
	"fmt"
	"maps"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// SequenceOfSteps is a step that executes many steps.
type SequenceOfSteps struct {
	resourceLoader cache.Cache
	runStep        func(ctx ctx.Context, globalCtx *GlobalContext, params *Params, specDefinition *proto.SpecDefinition) (*proto.StepResult, error)
}

func NewSequenceOfSteps(resourceLoader cache.Cache, runStep func(ctx ctx.Context, globalCtx *GlobalContext, params *Params, specDefinition *proto.SpecDefinition) (*proto.StepResult, error)) *SequenceOfSteps {
	return &SequenceOfSteps{
		resourceLoader: resourceLoader,
		runStep:        runStep,
	}
}

func (s *SequenceOfSteps) Run(ctx ctx.Context, stepsCtx *StepsContext, specDefinition *proto.SpecDefinition) (*proto.StepResult, error) {
	result := &proto.StepResult{
		SpecDefinition: specDefinition,
		Status:         proto.StepResult_failure,
		Outputs:        make(map[string]*structpb.Value),
		Exports:        make(map[string]string),
	}

	// Expand and add the definition environment to context
	err := addDefinitionEnv(stepsCtx, specDefinition.Definition)

	if err != nil {
		return result, fmt.Errorf("adding definition env: %w", err)
	}

	result.Env = stepsCtx.GetEnvs()

	// Create output and export files and add to context
	files, err := NewFiles(stepsCtx, specDefinition.Spec.Spec.OutputMethod, specDefinition.Spec.Spec.Outputs)

	if err != nil {
		return result, err
	}

	defer files.Cleanup()

	for _, step := range specDefinition.Definition.Steps {
		stepResult, err := s.runSubStep(ctx, stepsCtx, specDefinition, step)

		// Capture results even if there was an error
		if stepResult != nil {
			result.SubStepResults = append(result.SubStepResults, stepResult)

			if stepResult.Status == proto.StepResult_failure {
				return result, fmt.Errorf("failed step %q: %w", step.Name, err)
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

// runSubStep executes a single sub-step. The step reference inputs
// and environment are expanded. And the current environment is cloned
// into params in preparation for a recursive call to Run.
func (s *SequenceOfSteps) runSubStep(
	ctx ctx.Context,
	stepsCtx *StepsContext,
	specDefinition *proto.SpecDefinition,
	stepReference *proto.Step,
) (*proto.StepResult, error) {
	params := &Params{}

	// Load the step spec and definition from the cache
	subStepSpecDefinition, err := s.resourceLoader.Get(ctx, specDefinition.Dir, stepReference.Step)
	if err != nil {
		return nil, fmt.Errorf("getting step %q definition: %w", stepReference.Name, err)
	}

	params.Inputs = buildInputVars(stepReference, subStepSpecDefinition)

	for name, v := range params.Inputs {
		res, resErr := expression.Expand(stepsCtx, v.Value)

		if resErr != nil {
			return nil, fmt.Errorf("Cannot assign input %q due to error: %w", name, resErr)
		}

		err := params.Inputs[name].Assign(res)

		if err != nil {
			return nil, fmt.Errorf("Cannot assign input %q due to error: %w", name, err)
		}
	}

	// Clone environment and add step reference environment
	params.Env = maps.Clone(stepsCtx.Env)
	for k, v := range stepReference.Env {
		res, resErr := expression.ExpandString(stepsCtx, v)
		if resErr != nil {
			return nil, fmt.Errorf("Cannot assign env %q due to error: %s", k, resErr.Error())
		}
		params.Env[k] = res
	}

	// Run the step definition with the global context and expanded parameters
	result, err := s.runStep(ctx, stepsCtx.GlobalContext, params, subStepSpecDefinition)
	if err != nil {
		return result, err
	}

	// Record expanded step in results
	result.Step = &proto.Step{
		Name:   stepReference.Name,
		Step:   stepReference.Step,
		Inputs: mapValue(params.Inputs, func(v *context.Variable) *structpb.Value { return v.Value }),
		Env:    params.Env,
	}
	stepsCtx.Steps[stepReference.Name] = result
	return result, nil
}

func mapValue[Key comparable, Value any, NewValue any](value map[Key]Value, f func(v Value) NewValue) map[Key]NewValue {
	result := make(map[Key]NewValue, len(value))

	for k, v := range value {
		result[k] = f(v)
	}

	return result
}

func buildInputVars(stepReference *proto.Step, stepSpecDef *proto.SpecDefinition) map[string]*context.Variable {
	inputs := make(map[string]*context.Variable)

	for name, val := range stepReference.Inputs {
		inputs[name] = context.NewVariable(val, stepSpecDef.Spec.Spec.Inputs[name].Sensitive)
	}

	return inputs
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
