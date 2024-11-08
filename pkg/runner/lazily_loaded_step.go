package runner

import (
	ctx "context"
	"fmt"

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
	stepResource   StepResource
	workDir        string
}

func NewLazilyLoadedStep(globalCtx *GlobalContext, resourceLoader Cache, parser StepParser, stepReference *proto.Step, stepResource StepResource, workDir string) *LazilyLoadedStep {
	return &LazilyLoadedStep{
		globalCtx:      globalCtx,
		resourceLoader: resourceLoader,
		parser:         parser,
		stepReference:  stepReference,
		stepResource:   stepResource,
		workDir:        workDir,
	}
}

func (s *LazilyLoadedStep) Describe() string {
	return fmt.Sprintf("step %q", s.stepReference.Name)
}

// Run fetches a step definition, parses the step, and executes it.
// The step reference inputs and environment are expanded.
// The current environment is cloned into params in preparation for a recursive call to Run.
func (s *LazilyLoadedStep) Run(ctx ctx.Context, stepsCtx *StepsContext, globalCtx *GlobalContext, stepDir string, inputs map[string]*structpb.Value, env *Environment, steps map[string]*proto.StepResult) (*proto.StepResult, error) {
	parentStepsCtx, err := NewStepsContext(globalCtx, stepDir, inputs, env, steps)
	if err != nil {
		return nil, err
	}

	defer parentStepsCtx.Cleanup()

	step, params, subStepSpecDefinition, err := s.loadStep(ctx, parentStepsCtx, s.workDir)

	if err != nil {
		return nil, fmt.Errorf("failed to run %s: %w", s.Describe(), err)
	}

	subStepInputs := params.NewInputsWithDefault(subStepSpecDefinition.Spec.Spec.Inputs)
	result, err := step.Run(ctx, parentStepsCtx, globalCtx, subStepSpecDefinition.Dir, subStepInputs, env.AddLexicalScope(params.Env), map[string]*proto.StepResult{})

	if err != nil {
		return result, fmt.Errorf("failed to run %s: %w", s.Describe(), err)
	}

	return result, nil
}

func (s *LazilyLoadedStep) loadStep(ctx ctx.Context, stepsCtx *StepsContext, workingDir string) (Step, *Params, *proto.SpecDefinition, error) {
	stepResource, err := s.stepResource.Interpolate(stepsCtx.View())

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load: %w", err)
	}

	specDef, err := s.resourceLoader.Get(ctx, workingDir, stepResource)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load: %w", err)
	}

	inputs, err := buildInputVars(s.stepReference, specDef)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load: %w", err)
	}

	for name, v := range inputs {
		res, err := expression.Expand(stepsCtx.View(), v.Value)

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load: expand input %q: %w", name, err)
		}

		err = inputs[name].Assign(res)

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load: assign input %q: %w", name, err)
		}
	}

	env := map[string]string{}

	for k, v := range s.stepReference.Env {
		res, err := expression.ExpandString(stepsCtx.View(), v)

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load: env %q: %w", k, err)
		}

		env[k] = res
	}

	params := &Params{
		Inputs: inputs,
		Env:    stepsCtx.Env.AddLexicalScope(env).Values(),
	}

	step, err := s.parser.Parse(specDef, params, NewNamedStepReference(s.stepReference.Name, s.stepReference.Step))

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load: %w", err)
	}

	return step, params, specDef, nil
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
