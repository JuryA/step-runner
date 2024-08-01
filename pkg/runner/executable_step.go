package runner

import (
	ctx "context"
	"fmt"
	"os/exec"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/output"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// ExecutableStep is a step that executes a command.
type ExecutableStep struct {
}

func NewExecutableStep() *ExecutableStep {
	return &ExecutableStep{}
}

func (s *ExecutableStep) Run(
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
