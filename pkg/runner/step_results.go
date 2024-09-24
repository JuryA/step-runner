package runner

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepResults struct {
	values []*StepResult
}

func NewStepResults(values ...*StepResult) *StepResults {
	return &StepResults{values: values}
}

func (r *StepResults) FindOutputsForStepName(name string) (map[string]*structpb.Value, error) {
	for _, stepResult := range r.values {
		if hasStep, stepName := stepResult.StepName(); hasStep && stepName == name {
			return stepResult.Outputs(), nil
		}
	}

	return nil, fmt.Errorf("delegating outputs to %q: could not find substep", name)
}

func (r *StepResults) ToProtoStepResults() []*proto.StepResult {
	results := make([]*proto.StepResult, len(r.values))

	for i, stepResult := range r.values {
		results[i] = stepResult.ProtoStepResult()
	}

	return results
}
