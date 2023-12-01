package runner

import (
	ctx "context"
	"fmt"
	"os/exec"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/output"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Execution struct {
	defs      *cache.Definitions
}

func New(defs *cache.Definitions) (*Execution, error) {
	return &Execution{
		defs:      defs,
	}, nil
}

func (e *Execution) createContext(specDefinition *proto.StepDefinition, stepCall *proto.StepCall, globalCtx *context.Global) (*context.Steps, error) {
	stepsCtx := context.NewSteps()
	stepsCtx.Global = globalCtx
	stepsCtx.Env = stepCall.Env
	stepsCtx.Dir = specDefinition.Dir

	// Match inputs with definition
	for key, value := range specDefinition.Spec.Spec.Inputs {
		callValue := stepCall.Inputs[key]
		if value.Default == nil {
			if callValue == nil {
				return nil, fmt.Errorf("input %q required, but not defined", key)
			}
			stepsCtx.Inputs[key] = callValue
		} else {
			stepsCtx.Inputs[key] = value.Default
		}
	}

	// Reject invalid inputs
	for key, _ := range stepCall.Inputs {
		defValue := specDefinition.Spec.Spec.Inputs[key]
		if defValue == nil {
			return nil, fmt.Errorf("input %q not found", key)
		}
	}

	return stepsCtx, nil
}

func (e *Execution) Run(ctx ctx.Context, specDefinition *proto.StepDefinition, stepCall *proto.StepCall, globalCtx *context.Global) (*proto.StepResult, error) {
	stepsCtx, err := e.createContext(specDefinition, stepCall, globalCtx)
	if err != nil {
		return nil, err
	}

	var result *proto.StepResult

	switch specDefinition.Definition.Type {
	case proto.DefinitionType_exec:
		result, err = e.runExec(ctx, specDefinition.Definition.Exec, stepsCtx)

	case proto.DefinitionType_steps:
		result, err = e.runSteps(ctx, specDefinition.Definition.Steps, stepsCtx)

	default:
		err = fmt.Errorf("invalid type: %q", specDefinition.Definition.Type)
	}

	if result != nil {
		result.StepDefinition = specDefinition
	}
	return result, err
}

func (e *Execution) runExec(ctx ctx.Context, execDefinition *proto.Definition_Exec, stepsCtx *context.Steps) (*proto.StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("exec cancelled: %w", err)
	}

	files, err := output.New(stepsCtx)
	if err != nil {
		return nil, err
	}
	defer files.Cleanup()

	cmdArgs := []string{}
	for _, arg := range execDefinition.Command {
		expandedArg := expression.InterpolateString(stepsCtx, arg)
		cmdArgs = append(cmdArgs, expandedArg)
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	if execDefinition.WorkDir != "" {
		cmd.Dir = expression.InterpolateString(stepsCtx, execDefinition.WorkDir)
	} else {
		cmd.Dir = stepsCtx.Dir
	}
	// Only explicitly provided environment variables
	cmd.Env = stepsCtx.GetEnvList()
	// TODO: Use multi-writer
	cmd.Stdout = stepsCtx.Global.Stdout
	cmd.Stderr = stepsCtx.Global.Stderr
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("exec: %w", err)
	}
	exitCode := cmd.ProcessState.ExitCode()
	status := proto.StepResult_failure
	if exitCode == 0 {
		status = proto.StepResult_success
	}

	result := &proto.StepResult{
		Status:   status,
		ExitCode: int32(exitCode),
	}

	err = files.OutputTo(result)
	if err != nil {
		return nil, fmt.Errorf("outputting: %w", err)
	}
	err = files.ExportTo(stepsCtx.Global, result)
	if err != nil {
		return nil, fmt.Errorf("exporting: %w", err)
	}

	return result, nil
}

func (e *Execution) runSteps(ctx ctx.Context, stepsDefinition []*proto.Step, stepsCtx *context.Steps) (*proto.StepResult, error) {
	result := &proto.StepResult{}

	for _, step := range stepsDefinition {
		stepResult, err := e.runStep(ctx, step, stepsCtx)
		if err != nil {
			return nil, err
		}

		result.ChildrenStepResults = append(result.ChildrenStepResults, stepResult)
	}

	return result, nil
}

func (e *Execution) runStep(ctx ctx.Context, stepReference *proto.Step, stepsCtx *context.Steps) (*proto.StepResult, error) {
	stepCall := &proto.StepCall{}

	// Expand inputs
	stepCall.Inputs = make(map[string]*structpb.Value)
	for k, v := range stepReference.Inputs {
		stepCall.Inputs[k] = expression.InterpolateProtoValue(stepsCtx, v)
	}

	// Clone and expand env
	stepCall.Env = make(map[string]string)
	for k, v := range stepsCtx.Env {
		stepCall.Env[k] = v
	}
	for k, v := range stepReference.Env {
		stepCall.Env[k] = expression.InterpolateString(stepsCtx, v)
	}

	spec, def, dir, err := e.defs.Get(ctx, stepReference.Step)
	if err != nil {
		return nil, fmt.Errorf("getting step %q definition: %w", stepReference.Name, err)
	}

	// TODO: The `defs.Get` should return `proto.StepDefinition`
	stepDef := &proto.StepDefinition{
		Spec: spec,
		Definition: def,
		Dir: dir,
	}

	result, err := e.Run(ctx, stepDef, stepCall, stepsCtx.Global)
	if err != nil {
		return nil, err
	}

	result.Step = stepReference
	stepsCtx.Outputs[stepReference.Name] = result.Outputs
	return result, nil
}
