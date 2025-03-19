package runner

import (
	"fmt"
	"io"
	"os"

	"gitlab.com/gitlab-org/step-runner/pkg/precond"
)

type GlobalContext struct {
	WorkDir string
	Job     map[string]string
	Env     *Environment
	Stdout  io.Writer
	Stderr  io.Writer
}

func NewGlobalContext(env *Environment) *GlobalContext {
	globalCtx := &GlobalContext{
		Job:    map[string]string{},
		Env:    env,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	precond.MustNotBeNil(globalCtx.Env, "global context must have an environment")
	return globalCtx
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
