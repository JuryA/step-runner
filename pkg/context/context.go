package context

import (
	"io"
	"maps"
	"os"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Global struct {
	WorkDir string            `json:"work_dir"`
	Job     map[string]string `json:"job"`
	Env     map[string]string `json:"-"`
	Stdout  io.Writer         `json:"-"`
	Stderr  io.Writer         `json:"-"`
}

func NewGlobal() *Global {
	return &Global{
		Job:    map[string]string{},
		Env:    map[string]string{},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func (g *Global) InheritEnv(envs ...string) {
	if g.Env == nil {
		g.Env = make(map[string]string, len(envs))
	}
	for _, e := range envs {
		k, v, ok := strings.Cut(e, "=")
		if ok {
			g.Env[k] = v
		}
	}
}

type Steps struct {
	*Global

	StepDir    string                       `json:"step_dir"`
	OutputFile string                       `json:"output_file"`
	ExportFile string                       `json:"export_file"`
	Env        map[string]string            `json:"env"`
	Inputs     map[string]*structpb.Value   `json:"inputs"`
	Steps      map[string]*proto.StepResult `json:"steps"`
}

func NewSteps(global *Global) *Steps {
	return &Steps{
		Global: global,
		Env:    maps.Clone(global.Env),
		Inputs: map[string]*structpb.Value{},
		Steps:  map[string]*proto.StepResult{},
	}
}

func (s *Steps) GetEnvs() map[string]string {
	r := make(map[string]string)
	for k, v := range s.Global.Env {
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
