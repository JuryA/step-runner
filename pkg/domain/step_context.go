package domain

import (
	"maps"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepsCtx struct {
	*GlobalCtx

	StepDir    string                       `json:"step_dir"`
	OutputFile string                       `json:"output_file"`
	Env        map[string]string            `json:"env"`
	Inputs     map[string]*structpb.Value   `json:"inputs"`
	Steps      map[string]*proto.StepResult `json:"steps"`
}

func NewStepsCtx(global *GlobalCtx) *StepsCtx {
	return &StepsCtx{
		GlobalCtx: global,
		Env:       maps.Clone(global.Env),
		Inputs:    map[string]*structpb.Value{},
		Steps:     map[string]*proto.StepResult{},
	}
}

func (s *StepsCtx) GetEnvs() map[string]string {
	r := make(map[string]string)
	for k, v := range s.GlobalCtx.Env {
		r[k] = v
	}
	for k, v := range s.Env {
		r[k] = v
	}
	return r
}

func (s *StepsCtx) GetEnvList() []string {
	r := []string{}
	for k, v := range s.GetEnvs() {
		r = append(r, k+"="+v)
	}
	return r
}
