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
	subStepResults []*proto.StepResult
	stepsCtx       *StepsContext
}

func NewStepResultBuilder(loadedFrom StepReference, params *Params, specDef *proto.SpecDefinition, stepsCtx *StepsContext) *StepResultBuilder {
	return &StepResultBuilder{
		env:            make(map[string]string),
		execResult:     nil,
		exports:        make(map[string]string),
		loadedFrom:     loadedFrom,
		outputs:        make(map[string]*structpb.Value),
		params:         params,
		specDef:        specDef,
		subStepResults: make([]*proto.StepResult, 0),
		stepsCtx:       stepsCtx,
	}
}

func (bldr *StepResultBuilder) WithMergedOutputs(outputs map[string]*structpb.Value) *StepResultBuilder {
	maps.Copy(bldr.outputs, outputs)
	return bldr
}

func (bldr *StepResultBuilder) WithSubStepResult(result *proto.StepResult) *StepResultBuilder {
	if result != nil {
		bldr.subStepResults = append(bldr.subStepResults, result)
	}
	return bldr
}

func (bldr *StepResultBuilder) ObserveExecutedCmd(execResult *ExecResult, err error) error {
	if execResult != nil {
		bldr.execResult = execResult.ToProto()
	}
	return err
}

func (bldr *StepResultBuilder) ObserveOutputs(outputs map[string]*structpb.Value, delegateToResult *proto.StepResult, err error) error {
	bldr.WithMergedOutputs(outputs).WithSubStepResult(delegateToResult)
	return err
}

func (bldr *StepResultBuilder) ObserveExports(exports map[string]string, err error) error {
	bldr.exports = exports
	return err
}

func (bldr *StepResultBuilder) ObserveSubStepResult(stepResult *proto.StepResult, err error) (*proto.StepResult, error) {
	bldr.WithSubStepResult(stepResult)
	return stepResult, err
}

func (bldr *StepResultBuilder) ObserveMergedOutputs(outputs map[string]*structpb.Value, err error) error {
	bldr.WithMergedOutputs(outputs)
	return err
}

func (bldr *StepResultBuilder) BuildFailure() *proto.StepResult {
	return bldr.buildResult(proto.StepResult_failure)
}

func (bldr *StepResultBuilder) Build() *proto.StepResult {
	return bldr.buildResult(proto.StepResult_success)
}

func (bldr *StepResultBuilder) buildResult(status proto.StepResult_Status) *proto.StepResult {
	return &proto.StepResult{
		Step:           bldr.loadedFrom.ToProtoStep(bldr.params),
		SpecDefinition: bldr.specDef,
		Status:         status,
		Outputs:        bldr.outputs,
		Exports:        bldr.exports,
		Env:            bldr.stepsCtx.GetEnvs(),
		ExecResult:     bldr.execResult,
		SubStepResults: bldr.subStepResults,
	}
}
