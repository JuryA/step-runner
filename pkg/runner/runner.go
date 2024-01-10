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

type Execution struct {
	defs cache.Cache
}

type Params struct {
	Inputs map[string]*structpb.Value
	Env    map[string]string
}

func New(defs cache.Cache) (*Execution, error) {
	return &Execution{
		defs: defs,
	}, nil
}

func (e *Execution) createContext(specDefinition *proto.StepDefinition, params *Params, globalCtx *context.Global) (*context.Steps, error) {
	stepsCtx := context.NewSteps(globalCtx)
	maps.Copy(stepsCtx.Env, params.Env)
	stepsCtx.Dir = specDefinition.Dir

	// Match inputs with definition
	for key, value := range specDefinition.Spec.Spec.Inputs {
		callValue := params.Inputs[key]
		if callValue != nil {
			stepsCtx.Inputs[key] = callValue
		} else if value.Default != nil {
			stepsCtx.Inputs[key] = value.Default
		} else {
			return nil, fmt.Errorf("input %q required, but not defined", key)
		}
	}

	// Reject invalid inputs
	for key := range params.Inputs {
		defValue := specDefinition.Spec.Spec.Inputs[key]
		if defValue == nil {
			return nil, fmt.Errorf("input %q not found", key)
		}
	}

	return stepsCtx, nil
}

func (e *Execution) Run(ctx ctx.Context, specDefinition *proto.StepDefinition, params *Params, globalCtx *context.Global) (*proto.StepResult, error) {
	stepsCtx, err := e.createContext(specDefinition, params, globalCtx)
	if err != nil {
		return nil, err
	}

	result := &proto.StepResult{
		StepDefinition: specDefinition,
		Status:         proto.StepResult_success,
		Outputs:        make(map[string]string),
		Exports:        make(map[string]string),
	}

	switch specDefinition.Definition.Type {
	case proto.DefinitionType_exec:
		err = e.runExec(result, ctx, specDefinition.Definition.Exec, stepsCtx)

	case proto.DefinitionType_steps:
		err = e.runSteps(result, ctx, specDefinition.Definition.Steps, stepsCtx)

	default:
		err = fmt.Errorf("invalid type: %q", specDefinition.Definition.Type)
	}

	result.StepDefinition = specDefinition

	for k, v := range specDefinition.Definition.Outputs {
		res, resErr := expression.ExpandString(stepsCtx, v)
		if resErr == nil {
			result.Outputs[k] = res
		} else {
			fmt.Fprintf(stepsCtx.Global.Stderr, "Cannot assign %q due to error: %s", k, resErr.Error())
		}
	}
	return result, err
}

func (e *Execution) runExec(result *proto.StepResult, ctx ctx.Context, execDefinition *proto.Definition_Exec, stepsCtx *context.Steps) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("exec cancelled: %w", err)
	}

	files, err := output.New(stepsCtx)
	if err != nil {
		return err
	}
	defer files.Cleanup()

	cmdArgs := []string{}
	for _, arg := range execDefinition.Command {
		res, resErr := expression.ExpandString(stepsCtx, arg)
		if resErr != nil {
			return fmt.Errorf("Cannot interpolate command argument %q due to err: %s", arg, resErr.Error())
		}
		cmdArgs = append(cmdArgs, res)
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = stepsCtx.Dir
	if stepsCtx.Global.Dir != "" {
		res, resErr := expression.ExpandString(stepsCtx, stepsCtx.Global.Dir)
		if resErr != nil {
			return fmt.Errorf("cannot interpolate global context workdir %q due to err: %s", stepsCtx.Global.Dir, resErr.Error())
		}
		cmd.Dir = res
	} else if execDefinition.WorkDir != "" {
		res, resErr := expression.ExpandString(stepsCtx, execDefinition.WorkDir)
		if resErr != nil {
			return fmt.Errorf("Cannot interpolate command workdir %q due to err: %s", execDefinition.WorkDir, resErr.Error())
		}
		cmd.Dir = res
	}

	// Only explicitly provided environment variables
	cmd.Env = stepsCtx.GetEnvList()
	// TODO: Use multi-writer
	cmd.Stdout = stepsCtx.Global.Stdout
	cmd.Stderr = stepsCtx.Global.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if cmd.ProcessState.ExitCode() != 0 {
		result.ExitCode = int32(cmd.ProcessState.ExitCode())
		result.Status = proto.StepResult_failure
	}

	err = files.OutputTo(result)
	if err != nil {
		return fmt.Errorf("outputting: %w", err)
	}
	err = files.ExportTo(stepsCtx.Global, result)
	if err != nil {
		return fmt.Errorf("exporting: %w", err)
	}

	return nil
}

func (e *Execution) runSteps(result *proto.StepResult, ctx ctx.Context, stepsDefinition []*proto.Step, stepsCtx *context.Steps) error {
	for _, step := range stepsDefinition {
		stepResult, err := e.runStep(ctx, step, stepsCtx)
		if err != nil {
			return err
		}

		result.ChildrenStepResults = append(result.ChildrenStepResults, stepResult)

		// One step failed, return early
		if stepResult.Status == proto.StepResult_failure {
			result.Status = proto.StepResult_failure
			break
		}
	}

	return nil
}

func (e *Execution) runStep(ctx ctx.Context, stepReference *proto.Step, stepsCtx *context.Steps) (*proto.StepResult, error) {
	params := &Params{}

	// Expand inputs
	params.Inputs = make(map[string]*structpb.Value)
	for k, v := range stepReference.Inputs {
		res, resErr := expression.Expand(stepsCtx, v)
		if resErr != nil {
			return nil, fmt.Errorf("Cannot assign input %q due to error: %s", k, resErr.Error())
		}
		params.Inputs[k] = res
	}

	// Clone and expand env
	params.Env = make(map[string]string)
	for k, v := range stepsCtx.Env {
		params.Env[k] = v
	}
	for k, v := range stepReference.Env {
		res, resErr := expression.ExpandString(stepsCtx, v)
		if resErr != nil {
			return nil, fmt.Errorf("Cannot assign env %q due to error: %s", k, resErr.Error())
		}
		params.Env[k] = res
	}

	stepDefinition, err := e.defs.Get(ctx, stepReference.Step)
	if err != nil {
		return nil, fmt.Errorf("getting step %q definition: %w", stepReference.Name, err)
	}

	result, err := e.Run(ctx, stepDefinition, params, stepsCtx.Global)
	if err != nil {
		return nil, err
	}

	result.Step = stepReference
	stepsCtx.Steps[stepReference.Name] = result
	return result, nil
}
