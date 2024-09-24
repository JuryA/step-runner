package report

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

var StepResultsFile = "step-results.json"

type StepResultReport struct{}

func NewStepResultReport() *StepResultReport {
	return &StepResultReport{}
}

func (r *StepResultReport) Write(result *runner.StepResult) error {
	json, err := protojson.Marshal(result.ProtoStepResult())

	if err != nil {
		return fmt.Errorf("failed to write step results report: %w", err)
	}

	err = os.WriteFile(StepResultsFile, json, 0640)

	if err != nil {
		return fmt.Errorf("failed to write step results report: %w", err)
	}

	return nil
}
