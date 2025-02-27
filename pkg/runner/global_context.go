package runner

import (
	"fmt"
	"gitlab.com/gitlab-org/step-runner/proto"
	"io"
	"os"
)

type GlobalContext struct {
	WorkDir     string
	Job         map[string]string
	Attestation *proto.Attestation
	Env         *Environment
	Stdout      io.Writer
	Stderr      io.Writer
}

func NewGlobalContext(env *Environment) *GlobalContext {
	return &GlobalContext{
		Job: map[string]string{},
		Env: env,
		Attestation: &proto.Attestation{
			Enable:    false,
			Attestors: []string{},
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func (gc *GlobalContext) Logln(format string, v ...any) error {
	return gc.Logf(format+"\n", v...)
}

func (gc *GlobalContext) Logf(format string, v ...any) error {
	_, err := fmt.Fprintf(gc.Stdout, format, v...)
	if err != nil {
		return fmt.Errorf("writing to stdout: %w", err)
	}

	return nil
}
