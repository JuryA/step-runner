package runner

import (
	ctx "context"
	"fmt"
	"os/exec"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

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

	if err := result.ObserveEnv(stepsCtx.ExpandAndApplyEnv(s.specDef.Definition.Env)); err != nil {
		return result.BuildFailure(), fmt.Errorf("expand step env: %w", err)
	}

	if err := result.ObserveExecutedCmd(s.execCommand(ctx, stepsCtx)); err != nil {
		return result.BuildFailure(), fmt.Errorf("exec: %w", err)
	}

	if err := result.ObserveOutputs(s.readOutputs(stepsCtx.OutputFile)); err != nil {
		return result.BuildFailure(), fmt.Errorf("output file: %w", err)
	}

	exports, err := result.ObserveExports(stepsCtx.ExportFile.ReadEnvironment())
	if err != nil {
		return result.BuildFailure(), fmt.Errorf("export file: %w", err)
	}

	stepsCtx.GlobalContext.Env.Mutate(exports)
	return result.Build(), nil
}

func (s *ExecutableStep) execCommand(ctx ctx.Context, stepsCtx *StepsContext) (*ExecResult, error) {
	cmdArgs := []string{}

	for _, arg := range s.specDef.Definition.Exec.Command {
		res, err := expression.ExpandString(stepsCtx.View(), arg)

		if err != nil {
			return nil, fmt.Errorf("interpolate command argument %q: %w", arg, err)
		}

		cmdArgs = append(cmdArgs, res)
	}

	workDir, err := s.determineWorkDir(stepsCtx)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = workDir

	cmd.Env = stepsCtx.GetEnvList()
	cmd.Stdout = stepsCtx.GlobalContext.Stdout
	cmd.Stderr = stepsCtx.GlobalContext.Stderr

	err = cmd.Run()
	execResult := NewExecResult(cmd.Dir, cmd.Args, cmd.ProcessState.ExitCode())

	if err != nil {
		return execResult, err
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
		return "", fmt.Errorf("interpolate workdir %q: %w", workDir, err)
	}

	return res, nil
}

func (s *ExecutableStep) readOutputs(outputFile *StepFile) (map[string]*structpb.Value, error) {
	if s.specDef.Spec.Spec.OutputMethod == proto.OutputMethod_delegate {
		stepResult, err := outputFile.ReadStepResult()
		if err != nil {
			return nil, fmt.Errorf("delegate: %w", err)
		}

		return stepResult.Outputs, nil
	}

	return outputFile.ReadValues(s.specDef.Spec.Spec.Outputs)
}
