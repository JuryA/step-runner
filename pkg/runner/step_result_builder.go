package runner

import (
	"maps"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepResultBuilder struct {
	env            map[string]string
	execResult     *proto.StepResult_ExecResult
	exports        map[string]string
	loadedFrom     StepReference
	outputs        map[string]*structpb.Value
	params         *Params
	specDef        *proto.SpecDefinition
	subStepResults []*StepResult
}

func NewStepResultBuilder(loadedFrom StepReference, params *Params, specDef *proto.SpecDefinition) *StepResultBuilder {
	return &StepResultBuilder{
		env:            make(map[string]string),
		execResult:     nil,
		exports:        make(map[string]string),
		loadedFrom:     loadedFrom,
		outputs:        make(map[string]*structpb.Value),
		params:         params,
		specDef:        specDef,
		subStepResults: make([]*StepResult, 0),
	}
}

func (bldr *StepResultBuilder) WithExecResult(executedCmd *ExecResult) *StepResultBuilder {
	if executedCmd != nil {
		bldr.execResult = executedCmd.ToProto()
	}

	return bldr
}

func (bldr *StepResultBuilder) WithEnv(env map[string]string) *StepResultBuilder {
	bldr.env = env
	return bldr
}

func (bldr *StepResultBuilder) WithMergedOutputs(outputs map[string]*structpb.Value) *StepResultBuilder {
	maps.Copy(bldr.outputs, outputs)
	return bldr
}

func (bldr *StepResultBuilder) WithSubStepResult(result *StepResult) *StepResultBuilder {
	if result != nil {
		bldr.subStepResults = append(bldr.subStepResults, result)
	}
	return bldr
}

func (bldr *StepResultBuilder) WithExports(exports map[string]string) *StepResultBuilder {
	bldr.exports = exports
	return bldr
}

func (bldr *StepResultBuilder) BuildFailure() *StepResult {
	return bldr.buildResult(proto.StepResult_failure)
}

func (bldr *StepResultBuilder) Build() *StepResult {
	return bldr.buildResult(proto.StepResult_success)
}

func (bldr *StepResultBuilder) buildResult(status proto.StepResult_Status) *StepResult {
	protoStepResult := &proto.StepResult{
		Step:           bldr.loadedFrom.ToProtoStep(bldr.params),
		SpecDefinition: bldr.specDef,
		Status:         status,
		Outputs:        bldr.outputs,
		Exports:        bldr.exports,
		Env:            bldr.env,
		ExecResult:     bldr.execResult,
		SubStepResults: bldr.BuildSubStepResults().ToProtoStepResults(),
	}

	return NewStepResult(protoStepResult)
}

func (bldr *StepResultBuilder) BuildSubStepResults() *StepResults {
	return NewStepResults(bldr.subStepResults...)
}
