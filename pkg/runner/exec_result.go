package runner

import "gitlab.com/gitlab-org/step-runner/proto"

type ExecResult struct {
	workDir  string
	cmdArgs  []string
	exitCode int
}

func NewExecResult(workDir string, cmdArgs []string, exitCode int) *ExecResult {
	return &ExecResult{
		workDir:  workDir,
		cmdArgs:  cmdArgs,
		exitCode: exitCode,
	}
}

func (ec *ExecResult) ToProto() *proto.StepResult_ExecResult {
	return &proto.StepResult_ExecResult{
		Command:  ec.cmdArgs,
		WorkDir:  ec.workDir,
		ExitCode: int32(ec.exitCode),
	}
}
