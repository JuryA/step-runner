package expression

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

// InterpolationContext contains fields that can be accessed by expressions.
type InterpolationContext struct {
	Context     *proto.StepsContext        `json:"steps_context"`
	Env         map[string]string          `json:"env"`
	ExportFile  string                     `json:"export_file"`
	Inputs      map[string]*structpb.Value `json:"inputs"`
	Job         map[string]string          `json:"job"`
	OutputFile  string                     `json:"output_file"`
	StepDir     string                     `json:"step_dir"`
	StepResults map[string]*StepResultView `json:"steps"`
	WorkDir     string                     `json:"work_dir"`
}

type StepResultView struct {
	Outputs map[string]*structpb.Value `json:"outputs"`
}
