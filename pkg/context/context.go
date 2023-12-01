package context

import (
	"io"
	"os"

	"google.golang.org/protobuf/types/known/structpb"
)

type Global struct {
	Job map[string]string
	Env map[string]string
	Stdout io.Writer
	Stderr io.Writer
}

func NewGlobal() *Global {
	return &Global{
		Job: map[string]string{},
		Env: map[string]string{},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func (g *Global) GetMatches() map[string]*structpb.Value {
	m := make(map[string]*structpb.Value)
	for k, v := range g.Job {
		m["job." + k] = structpb.NewStringValue(v)
	}
	return m
}

type Steps struct {
	Global *Global

	Dir string
	Env map[string]string
	Inputs map[string]*structpb.Value
	Outputs map[string]map[string]string
}

func NewSteps() *Steps {
	return &Steps{
		Env: map[string]string{},
		Inputs: map[string]*structpb.Value{},
		Outputs: map[string]map[string]string{},
	}
}

func (s *Steps) GetMatches() map[string]*structpb.Value {
	m := make(map[string]*structpb.Value)
	for k, v := range s.Global.GetMatches() {
		m[k] = v
	}
	for name, value := range s.Inputs {
		m["inputs." + name] = value
	}
	for step, outputs := range s.Outputs {
		for name, value := range outputs {
			m["steps." + step + ".outputs." + name] = structpb.NewStringValue(value)
		}
	}
	return m
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
	r := []string{};
	for k, v := range s.GetEnvs() {
		r = append(r, k+"="+v)
	}
	return r
}

