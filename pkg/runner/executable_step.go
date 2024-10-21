package runner

import (
	ctx "context"
	"fmt"
	"os/exec"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// ExecutableStep is a step that executes a command.
type ExecutableStep struct {
	loadedFrom StepReference
	params     *Params
	specDef    *proto.SpecDefinition
}

func NewExecutableStep(loadedFrom StepReference, params *Params, specDef *proto.SpecDefinition) *ExecutableStep {
	return &ExecutableStep{
		loadedFrom: loadedFrom,
		params:     params,
		specDef:    specDef,
	}
}

func (s *ExecutableStep) Describe() string {
	return fmt.Sprintf("executable step %q", strings.Join(s.specDef.Definition.Exec.Command, " "))
}

func (s *ExecutableStep) Run(ctx ctx.Context, stepsCtx *StepsContext) (*proto.StepResult, error) {
	result := NewStepResultBuilder(s.loadedFrom, s.params, s.specDef)
	files, err := NewFiles(stepsCtx, s.specDef.Spec.Spec.OutputMethod, s.specDef.Spec.Spec.Outputs)

	if err != nil {
		return result.BuildFailure(), err
	}

	defer files.Cleanup()

	err = stepsCtx.ExpandAndApplyEnv(s.specDef.Definition.Env)
	result.WithEnv(stepsCtx.GetEnvs())

	if err != nil {
		return result.BuildFailure(), fmt.Errorf("failed to run executable step: %w", err)
	}

	executedCmd, err := s.execCommand(ctx, stepsCtx)
	result.WithExecResult(executedCmd)

	if err != nil {
		return result.BuildFailure(), err
	}

	outputs, delegateToResult, err := files.Outputs()
	result.WithMergedOutputs(outputs).WithSubStepResult(delegateToResult)

	if err != nil {
		return result.BuildFailure(), fmt.Errorf("failed to run executable step: %w", err)
	}

	exports, err := stepsCtx.GlobalContext.Exports()
	result.WithExports(exports)

	if err != nil {
		return result.BuildFailure(), fmt.Errorf("failed to run executable step: %w", err)
	}

	return result.Build(), nil
}

func (s *ExecutableStep) execCommand(ctx ctx.Context, stepsCtx *StepsContext) (*ExecResult, error) {
	cmdArgs := []string{}

	for _, arg := range s.specDef.Definition.Exec.Command {
		res, err := expression.ExpandString(stepsCtx.View(), arg)

		if err != nil {
			return nil, fmt.Errorf("failed to interpolate command argument %q: %w", arg, err)
		}

		cmdArgs = append(cmdArgs, res)
	}

	workDir, err := s.determineWorkDir(stepsCtx)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = workDir

	// Provide only environment variables from the steps context. Not from the step runner's environment.
	cmd.Env = stepsCtx.GetEnvList()
	// TODO: Use multi-writer
	cmd.Stdout = stepsCtx.GlobalContext.Stdout
	cmd.Stderr = stepsCtx.GlobalContext.Stderr

	err = cmd.Run()
	execResult := NewExecResult(cmd.Dir, cmd.Args, cmd.ProcessState.ExitCode())

	if err != nil {
		return execResult, fmt.Errorf("exec: %w", err)
	}

	return execResult, nil
}

func (s *ExecutableStep) determineWorkDir(stepsCtx *StepsContext) (string, error) {
	workDir := s.specDef.Definition.Exec.WorkDir

	if workDir == "" {
		return stepsCtx.WorkDir, nil
	}

	res, err := expression.ExpandString(stepsCtx.View(), workDir)

	if err != nil {
		return "", fmt.Errorf("failed to interpolate workdir %q: %w", workDir, err)
	}

	return res, nil
}
