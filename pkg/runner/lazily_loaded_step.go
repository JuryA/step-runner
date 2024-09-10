package runner

import (
	ctx "context"
	"fmt"
	"maps"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// LazilyLoadedStep is a step that dynamically fetches, parses and executes a step definition.
type LazilyLoadedStep struct {
	globalCtx      *GlobalContext
	resourceLoader Cache
	parser         StepParser
	stepReference  *proto.Step
}

func NewLazilyLoadedStep(globalCtx *GlobalContext, resourceLoader Cache, parser StepParser, stepReference *proto.Step) *LazilyLoadedStep {
	return &LazilyLoadedStep{
		globalCtx:      globalCtx,
		resourceLoader: resourceLoader,
		parser:         parser,
		stepReference:  stepReference,
	}
}

func (s *LazilyLoadedStep) Describe() string {
	return fmt.Sprintf("lazily-evaluated step %q", s.stepReference.Name)
}

// Run fetches a step definition, parses the step, and executes it.
// The step reference inputs and environment are expanded.
// The current environment is cloned into params in preparation for a recursive call to Run.
func (s *LazilyLoadedStep) Run(ctx ctx.Context, parentStepsCtx *StepsContext, specDefinition *proto.SpecDefinition) (*proto.StepResult, error) {
	step, params, subStepSpecDefinition, err := s.loadStep(ctx, parentStepsCtx, specDefinition.Dir)

	if err != nil {
		return nil, fmt.Errorf("failed to run %s: %w", s.Describe(), err)
	}

	env := s.globalCtx.NewEnvMergedFrom(params.Env)
	inputs := params.NewInputsWithDefault(subStepSpecDefinition.Spec.Spec.Inputs)
	stepsCtx := NewStepsContext(s.globalCtx, subStepSpecDefinition.Dir, inputs, env)

	result, err := step.Run(ctx, stepsCtx, subStepSpecDefinition)

	if err != nil {
		return result, fmt.Errorf("failed to run %s: %w", s.Describe(), err)
	}

	// Record expanded step in results
	result.Step = &proto.Step{
		Name:   s.stepReference.Name,
		Step:   s.stepReference.Step,
		Inputs: mapValue(params.Inputs, func(v *context.Variable) *structpb.Value { return v.Value }),
		Env:    params.Env,
	}

	return result, nil
}

func (s *LazilyLoadedStep) loadStep(ctx ctx.Context, stepsCtx *StepsContext, workingDir string) (Step, *Params, *proto.SpecDefinition, error) {
	specDef, err := s.resourceLoader.Get(ctx, workingDir, s.stepReference.Step)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load: %w", err)
	}

	inputs, err := buildInputVars(s.stepReference, specDef)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load: %w", err)
	}

	for name, v := range inputs {
		res, err := expression.Expand(stepsCtx, v.Value)

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load: expand input %q: %w", name, err)
		}

		err = inputs[name].Assign(res)

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load: assign input %q: %w", name, err)
		}
	}

	// Clone environment and add step reference environment
	env := maps.Clone(stepsCtx.Env)

	for k, v := range s.stepReference.Env {
		res, err := expression.ExpandString(stepsCtx, v)

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load: env %q: %w", k, err)
		}

		env[k] = res
	}

	params := &Params{
		Inputs: inputs,
		Env:    env,
	}

	step, err := s.parser.Parse(specDef, params)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load: %w", err)
	}

	return step, params, specDef, nil
}

func mapValue[Key comparable, Value any, NewValue any](value map[Key]Value, f func(v Value) NewValue) map[Key]NewValue {
	result := make(map[Key]NewValue, len(value))

	for k, v := range value {
		result[k] = f(v)
	}

	return result
}

func buildInputVars(stepReference *proto.Step, stepSpecDef *proto.SpecDefinition) (map[string]*context.Variable, error) {
	inputs := make(map[string]*context.Variable)

	for name, val := range stepReference.Inputs {
		input, ok := stepSpecDef.Spec.Spec.Inputs[name]

		if !ok {
			return inputs, fmt.Errorf("step does not accept input with name %q", name)
		}

		inputs[name] = context.NewVariable(val, input.Sensitive)
	}

	return inputs, nil
}
