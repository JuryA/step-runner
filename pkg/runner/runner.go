package runner

import (
	ctx "context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// Execution is the execution of a single step.
type Execution struct {
	defs cache.Cache
}

// Params are the input and environment parameters for an execution.
type Params struct {
	Inputs map[string]*context.Variable
	Env    map[string]string
}

// New creates a new execution using a shared cache.
func New(defs cache.Cache) (*Execution, error) {
	return &Execution{
		defs: defs,
	}, nil
}

// Run begins execution of a single step, recursively executing
// sub-steps until completion. The step results returned will be a
// tree structure in the shape of the step execution and its
// sub-steps.
//
// The step inherits the environment variables of the global
// context. Environment variables provided in the params will shadow
// those from the global context. And environment variables in the
// step's definition will shadow those provided in the params and the
// globals. However when Run returns only those environment variables
// exported by steps to the global context will remain.
//
// Step inputs are given in params. They are combined with the step
// spec which provides defaults and constraints on the valid set and
// type of inputs. Inputs then become available in the step context
// for other values to reference through expressions.
//
// Inputs and environment variables in the params are assumed to be
// already expanded (no expressions will be evaluated). Values in step
// definitions (environment variables, input values, commands, etc...)
// will be expanded before sub-steps are executed.
func (e *Execution) Run(
	ctx ctx.Context,
	globalCtx *GlobalContext,
	params *Params,
	specDefinition *proto.SpecDefinition,
) (*proto.StepResult, error) {
	if err := validateInputs(specDefinition.Spec, params.Inputs); err != nil {
		return &proto.StepResult{Status: proto.StepResult_failure}, err
	}

	if specDefinition.Definition.Type != proto.DefinitionType_exec && specDefinition.Definition.Type != proto.DefinitionType_steps {
		return &proto.StepResult{Status: proto.StepResult_failure}, fmt.Errorf("invalid type: %q", specDefinition.Definition.Type)
	}

	env := globalCtx.NewEnvMergedFrom(params.Env)
	inputs := e.valueOrDefault(params.Inputs, specDefinition.Spec.Spec.Inputs)
	stepsCtx := NewStepsContext(globalCtx, specDefinition.Dir, inputs, env)

	if specDefinition.Definition.Type == proto.DefinitionType_exec {
		return NewExecutableStep().Run(ctx, stepsCtx, specDefinition)
	}

	return NewSequenceOfSteps(e.defs, e.Run).Run(ctx, stepsCtx, specDefinition)
}

func validateInputs(spec *proto.Spec, inputs map[string]*context.Variable) error {
	for key, value := range spec.Spec.Inputs {
		if inputs[key] == nil && value.Default == nil {
			return fmt.Errorf("input %q required, but not defined", key)
		}
	}

	for key := range inputs {
		if spec.Spec.Inputs[key] == nil {
			return fmt.Errorf("input %q not found", key)
		}
	}

	return nil
}

func (e *Execution) valueOrDefault(inputs map[string]*context.Variable, specInputs map[string]*proto.Spec_Content_Input) map[string]*structpb.Value {
	newInputs := make(map[string]*structpb.Value)

	for key, value := range specInputs {
		if inputs[key] != nil {
			newInputs[key] = inputs[key].Value
		} else {
			newInputs[key] = value.Default
		}
	}

	return newInputs
}
