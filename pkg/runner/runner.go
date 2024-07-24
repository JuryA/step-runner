package runner

import (
	ctx "context"
	"fmt"
	"maps"
	"os/exec"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/output"
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
	globalCtx *context.Global,
	params *Params,
	specDefinition *proto.SpecDefinition,
) (*proto.StepResult, error) {
	stepsCtx := context.NewSteps(globalCtx)

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

	result := &proto.StepResult{
		SpecDefinition: specDefinition,
		Status:         proto.StepResult_success,
		Outputs:        make(map[string]*structpb.Value),
		Exports:        make(map[string]string),
	}

	switch specDefinition.Definition.Type {
	case proto.DefinitionType_exec:
		err = e.runExec(ctx, stepsCtx, specDefinition, result)

	case proto.DefinitionType_steps:
		err = e.runSteps(ctx, stepsCtx, specDefinition, result)

	default:
		err = fmt.Errorf("invalid type: %q", specDefinition.Definition.Type)
	}
	if err != nil {
		// We return partial results with an error to help
		// callers understand what went wrong.
		result.Status = proto.StepResult_failure
		return result, err
	}

	result.SpecDefinition = specDefinition

	return result, err
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

// addInputs combines the provided input parameters with the step
// spec. Missing inputs are given defaults. Missing inputs without a
// default produce an error. Extra inputs not declared also produce an
// error.
func addInputs(stepsCtx *context.Steps, spec *proto.Spec, inputs map[string]*context.Variable) error {
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
func addDefinitionEnv(stepsCtx *context.Steps, definition *proto.Definition) error {
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

// runExec executes an exec type step. The exec command and working
// directory are expanded with the step context and the result is
// written to the provided step result.
func (e *Execution) runExec(
	ctx ctx.Context,
	stepsCtx *context.Steps,
	specDefinition *proto.SpecDefinition,
	result *proto.StepResult,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("exec cancelled: %w", err)
	}

	result.ExecResult = &proto.StepResult_ExecResult{}

	execDefinition := specDefinition.Definition.Exec
	outputs := specDefinition.Spec.Spec.Outputs
	outputMethod := specDefinition.Spec.Spec.OutputMethod

	// Create output and export files and add to context
	files, err := output.New(stepsCtx, outputMethod, outputs)
	if err != nil {
		return err
	}
	defer files.Cleanup()

	// Expand and add the definition environment to context
	err = addDefinitionEnv(stepsCtx, specDefinition.Definition)
	if err != nil {
		return fmt.Errorf("adding definition env: %w", err)
	}

	// Expand args
	cmdArgs := []string{}
	for _, arg := range execDefinition.Command {
		res, resErr := expression.ExpandString(stepsCtx, arg)
		if resErr != nil {
			return fmt.Errorf("Cannot interpolate command argument %q due to err: %s", arg, resErr.Error())
		}
		cmdArgs = append(cmdArgs, res)
	}
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	result.ExecResult.Command = cmd.Args

	// Expand working directory if present. Otherwise fall back to
	// the working directory defined globally.
	if execDefinition.WorkDir != "" {
		res, resErr := expression.ExpandString(stepsCtx, execDefinition.WorkDir)
		if resErr != nil {
			return fmt.Errorf("Cannot interpolate command workdir %q due to err: %s", execDefinition.WorkDir, resErr.Error())
		}
		cmd.Dir = res
	} else {
		cmd.Dir = stepsCtx.WorkDir
	}
	result.ExecResult.WorkDir = cmd.Dir

	// Provide only environment variables from the steps
	// context. Not from the step runner's environment.
	cmd.Env = stepsCtx.GetEnvList()
	result.Env = stepsCtx.GetEnvs()
	// TODO: Use multi-writer
	cmd.Stdout = stepsCtx.Global.Stdout
	cmd.Stderr = stepsCtx.Global.Stderr

	// Capture results of execution
	err = cmd.Run()
	result.ExecResult.ExitCode = int32(cmd.ProcessState.ExitCode())
	if result.ExecResult.ExitCode != 0 {
		result.Status = proto.StepResult_failure
	}
	if err != nil {
		return fmt.Errorf("exec: %w, ", err)
	}

	err = files.OutputTo(result)
	if err != nil {
		return fmt.Errorf("outputting: %w", err)
	}
	err = stepsCtx.Global.ExportTo(result)
	if err != nil {
		return fmt.Errorf("exporting: %w", err)
	}

	return nil
}

// runSteps executes an steps type step. Each sub-step's environment
// and inputs are expanded with the step context and the result is
// written to the provided step result.
func (e *Execution) runSteps(
	ctx ctx.Context,
	stepsCtx *context.Steps,
	specDefinition *proto.SpecDefinition,
	result *proto.StepResult,
) error {
	// Expand and add the definition environment to context
	err := addDefinitionEnv(stepsCtx, specDefinition.Definition)
	if err != nil {
		return fmt.Errorf("adding definition env: %w", err)
	}
	result.Env = stepsCtx.GetEnvs()

	// Create output and export files and add to context
	files, err := output.New(stepsCtx, specDefinition.Spec.Spec.OutputMethod, specDefinition.Spec.Spec.Outputs)
	if err != nil {
		return err
	}
	defer files.Cleanup()

	result.Status = proto.StepResult_success
	for _, protoStep := range specDefinition.Definition.Steps {
		step, err := e.loadStep(ctx, specDefinition.Dir, protoStep)

		if err != nil {
			return fmt.Errorf("failed to run steps due to: %w", err)
		}

		stepResult, err := e.runSubStep(ctx, stepsCtx, step)

		// Capture results even if there was an error
		if stepResult != nil {
			result.SubStepResults = append(result.SubStepResults, stepResult)

			// If a sub-step fails then fail this step
			if stepResult.Status == proto.StepResult_failure {
				result.Status = proto.StepResult_failure
				return fmt.Errorf("failed step %q: %w", step.Name(), err)
			}
		}
		if err != nil {
			return err
		}
	}

	// Delegate outputs are surfaced directly, effectively making
	// the delegation mechanism "disappear" from the execution
	// context.
	if specDefinition.Spec.Spec.OutputMethod == proto.OutputMethod_delegate {
		return mergeDelegateOutput(specDefinition.Definition.Delegate, result)
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
			fmt.Fprintf(stepsCtx.Global.Stderr, "Cannot assign %q due to error: %s", k, resErr.Error())
		}
	}

	return nil
}

// runSubStep executes a single sub-step. The step reference inputs
// and environment are expanded. And the current environment is cloned
// into params in preparation for a recursive call to Run.
func (e *Execution) runSubStep(ctx ctx.Context, stepsCtx *context.Steps, step *context.Step) (*proto.StepResult, error) {
	inputs, err := step.ExpandInputs(stepsCtx, expression.Expand)

	if err != nil {
		return nil, fmt.Errorf("failed to run step %q: %w", step.Name(), err)
	}

	params := &Params{
		Inputs: inputs,
	}

	// Clone environment and add step reference environment
	params.Env = maps.Clone(stepsCtx.Env)
	for k, v := range step.Env() {
		res, resErr := expression.ExpandString(stepsCtx, v)
		if resErr != nil {
			return nil, fmt.Errorf("Cannot assign env %q due to error: %s", k, resErr.Error())
		}
		params.Env[k] = res
	}

	// Run the step definition with the global context and expanded parameters
	result, err := e.Run(ctx, stepsCtx.Global, params, step.ProtoDef)
	if err != nil {
		return result, err
	}

	// Record expanded step in results
	result.Step = &proto.Step{
		Name:   step.Name(),
		Step:   step.ProtoStep.Step,
		Inputs: mapValue(params.Inputs, func(v *context.Variable) *structpb.Value { return v.Value }),
		Env:    params.Env,
	}
	stepsCtx.Steps[step.Name()] = result
	return result, nil
}

func mapValue[Key comparable, Value any, NewValue any](value map[Key]Value, f func(v Value) NewValue) map[Key]NewValue {
	result := make(map[Key]NewValue, len(value))

	for k, v := range value {
		result[k] = f(v)
	}

	return result
}

func (e *Execution) loadStep(ctx ctx.Context, parentDir string, step *proto.Step) (*context.Step, error) {
	specDefinition, err := e.defs.Get(ctx, parentDir, step.Step)

	if err != nil {
		return nil, fmt.Errorf("failed to load step %q due to: %w", step.Name, err)
	}

	inputs := make(map[string]*context.Variable)

	for name, val := range step.Inputs {
		inputs[name] = context.NewVariable(val, specDefinition.Spec.Spec.Inputs[name].Sensitive)
	}

	return context.NewStep(step, specDefinition, inputs), nil
}
