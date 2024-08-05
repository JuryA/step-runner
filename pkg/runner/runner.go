package runner

import (
	ctx "context"
	"fmt"
	"maps"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
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
	stepsCtx := NewStepsContext(globalCtx)

	// We tell steps where to find their cached definition so they
	// can find their files. And so that sub-steps with relative
	// references know where to start.
	stepsCtx.StepDir = specDefinition.Dir

	// Add param inputs and environment to context
	err := addInputs(stepsCtx, specDefinition.Spec, params.Inputs)
	if err != nil {
		return nil, fmt.Errorf("adding inputs: %w", err)
	}
	maps.Copy(stepsCtx.Env, params.Env)

	switch specDefinition.Definition.Type {
	case proto.DefinitionType_exec:
		return NewExecutableStep().Run(ctx, stepsCtx, specDefinition)

	case proto.DefinitionType_steps:
		return NewSequenceOfSteps(e.defs, e.Run).Run(ctx, stepsCtx, specDefinition)
	}

	result := &proto.StepResult{SpecDefinition: specDefinition, Status: proto.StepResult_failure}
	return result, fmt.Errorf("invalid type: %q", specDefinition.Definition.Type)
}

// addInputs combines the provided input parameters with the step
// spec. Missing inputs are given defaults. Missing inputs without a
// default produce an error. Extra inputs not declared also produce an
// error.
func addInputs(stepsCtx *StepsContext, spec *proto.Spec, inputs map[string]*context.Variable) error {
	// Match inputs with definition
	for key, value := range spec.Spec.Inputs {
		callValue := inputs[key]
		if callValue != nil {
			stepsCtx.Inputs[key] = callValue.Value
		} else if value.Default != nil {
			stepsCtx.Inputs[key] = value.Default
		} else {
			return fmt.Errorf("input %q required, but not defined", key)
		}
	}

	// Reject invalid inputs
	for key := range inputs {
		defValue := spec.Spec.Inputs[key]
		if defValue == nil {
			return fmt.Errorf("input %q not found", key)
		}
	}

	return nil
}

// addDefinitionEnv expands the step definition environment variables
// with the step context. After expansion, definition environment
// variables are added to the step context.
func addDefinitionEnv(stepsCtx *StepsContext, definition *proto.Definition) error {
	defEnv := map[string]string{}
	for k, v := range definition.Env {
		res, resErr := expression.ExpandString(stepsCtx, v)
		if resErr != nil {
			return fmt.Errorf("Cannot assign env %q due to error: %s", k, resErr.Error())
		}
		defEnv[k] = res
	}
	maps.Copy(stepsCtx.Env, defEnv)
	return nil
}
