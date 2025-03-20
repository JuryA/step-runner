package runner

import (
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
		subStepResults: make([]*proto.StepResult, 0),
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

func (bldr *StepResultBuilder) WithOutputs(outputs map[string]*structpb.Value) *StepResultBuilder {
	bldr.outputs = outputs
	return bldr
}

func (bldr *StepResultBuilder) WithSubStepResult(result *proto.StepResult) *StepResultBuilder {
	if result != nil {
		bldr.subStepResults = append(bldr.subStepResults, result)
	}
	return bldr
}

func (bldr *StepResultBuilder) WithExports(exports *Environment) *StepResultBuilder {
	if exports != nil {
		bldr.exports = exports.Values()
	}
	return bldr
}

func (bldr *StepResultBuilder) ObserveEnv(env *Environment, err error) error {
	if env != nil {
		bldr.WithEnv(env.Values())
	}
	return err
}

func (bldr *StepResultBuilder) ObserveExecutedCmd(execResult *ExecResult, err error) error {
	bldr.WithExecResult(execResult)
	return err
}

func (bldr *StepResultBuilder) ObserveOutputs(outputs map[string]*structpb.Value, err error) error {
	bldr.WithOutputs(outputs)
	return err
}

func (bldr *StepResultBuilder) ObserveExports(exports *Environment, err error) (*Environment, error) {
	bldr.WithExports(exports)
	return exports, err
}

func (bldr *StepResultBuilder) ObserveStepResult(stepResult *proto.StepResult, err error) (*proto.StepResult, error) {
	bldr.WithSubStepResult(stepResult)
	return stepResult, err
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
		Env:            bldr.env,
		ExecResult:     bldr.execResult,
		SubStepResults: bldr.subStepResults,
	}
}
