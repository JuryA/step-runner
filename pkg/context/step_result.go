package context

import (
	"maps"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func NewStepResult(options ...func(*proto.StepResult)) *proto.StepResult {
	stepResult := &proto.StepResult{
		Status:  proto.StepResult_unspecified,
		Outputs: make(map[string]*structpb.Value),
		Exports: make(map[string]string),
	}

	for _, opt := range options {
		opt(stepResult)
	}

	return stepResult
}

func NewFailedStepResult(options ...func(*proto.StepResult)) *proto.StepResult {
	defaults := []func(*proto.StepResult){WithStepResultFailureStatus()}
	return NewStepResult(append(defaults, options...)...)
}

func WithStepResultSpecDefinition(specDef *proto.SpecDefinition) func(*proto.StepResult) {
	return func(stepResult *proto.StepResult) {
		stepResult.SpecDefinition = specDef
	}
}

func WithStepResultFailureStatus() func(*proto.StepResult) {
	return func(stepResult *proto.StepResult) {
		stepResult.Status = proto.StepResult_failure
	}
}

func WithStepResultEnv(env map[string]string) func(*proto.StepResult) {
	return func(stepResult *proto.StepResult) {
		stepResult.Env = env
	}
}

func WithStepResultOutputs(outputs map[string]*structpb.Value) func(*proto.StepResult) {
	return func(stepResult *proto.StepResult) {
		stepResult.Outputs = outputs
	}
}

func WithStepResultAdditionalOutputs(outputs map[string]*structpb.Value) func(*proto.StepResult) {
	return func(stepResult *proto.StepResult) {
		if stepResult.Outputs == nil {
			stepResult.Outputs = map[string]*structpb.Value{}
		}

		maps.Copy(stepResult.Outputs, outputs)
	}
}

func WithStepResultSubStepResult(subStepResult *proto.StepResult) func(*proto.StepResult) {
	return func(stepResult *proto.StepResult) {
		stepResult.SubStepResults = append(stepResult.SubStepResults, subStepResult)
	}
}

func WithStepResultExecResultOpts(opts ...func(*proto.StepResult_ExecResult)) func(*proto.StepResult) {
	return func(stepResult *proto.StepResult) {
		execResult := &proto.StepResult_ExecResult{}

		for _, opt := range opts {
			opt(execResult)
		}

		stepResult.ExecResult = execResult
	}
}

func WithExecResultWorkDir(workDir string) func(*proto.StepResult_ExecResult) {
	return func(execResult *proto.StepResult_ExecResult) {
		execResult.WorkDir = workDir
	}
}

func WithExecResultExitCode(exitCode int) func(*proto.StepResult_ExecResult) {
	return func(execResult *proto.StepResult_ExecResult) {
		execResult.ExitCode = int32(exitCode)
	}
}

func WithExecResultCmd(cmd []string) func(*proto.StepResult_ExecResult) {
	return func(execResult *proto.StepResult_ExecResult) {
		execResult.Command = cmd
	}
}
