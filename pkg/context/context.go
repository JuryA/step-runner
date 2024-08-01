package context

import (
	"maps"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Steps struct {
	*runner.GlobalContext

	StepDir    string                       `json:"step_dir"`
	OutputFile string                       `json:"output_file"`
	Env        map[string]string            `json:"env"`
	Inputs     map[string]*structpb.Value   `json:"inputs"`
	Steps      map[string]*proto.StepResult `json:"steps"`
}

func NewSteps(global *runner.GlobalContext) *Steps {
	return &Steps{
		GlobalContext: global,
		Env:           maps.Clone(global.Env),
		Inputs:        map[string]*structpb.Value{},
		Steps:         map[string]*proto.StepResult{},
	}
}

func (s *Steps) GetEnvs() map[string]string {
	r := make(map[string]string)
	for k, v := range s.GlobalContext.Env {
		r[k] = v
	}
	for k, v := range s.Env {
		r[k] = v
	}
	return r
}

func (s *Steps) GetEnvList() []string {
	r := []string{}
	for k, v := range s.GetEnvs() {
		r = append(r, k+"="+v)
	}
	return r
}
