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

