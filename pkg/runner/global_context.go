package runner

import (
	"fmt"
	"io"

	"gitlab.com/gitlab-org/step-runner/pkg/precond"
)

type GlobalContext struct {
	workDir string
	job     map[string]string
	env     *Environment
	stdout  io.Writer
	stderr  io.Writer
}

func NewGlobalContext(workDir string, job map[string]string, env *Environment, stdout, stderr io.Writer) *GlobalContext {
	precond.MustNotBeNil(env, "global context must have an environment")

	return &GlobalContext{
		workDir: workDir,
		job:     job,
		env:     env,
		stdout:  stdout,
		stderr:  stderr,
	}
}

func (gc *GlobalContext) WorkDir() string {
	return gc.workDir
}

func (gc *GlobalContext) Job() map[string]string {
	return gc.job
}

func (gc *GlobalContext) Env() *Environment {
	return gc.env
}

func (gc *GlobalContext) EnvWithLexicalScope(envVars map[string]string) *Environment {
	return gc.env.AddLexicalScope(envVars)
}

func (gc *GlobalContext) AddGlobalEnv(env *Environment) {
	gc.env.Mutate(env)
}

func (gc *GlobalContext) Pipe() (io.Writer, io.Writer) {
	return gc.stdout, gc.stderr
}

func (gc *GlobalContext) Logln(format string, v ...any) error {
	return gc.Logf(format+"\n", v...)
}

func (gc *GlobalContext) Logf(format string, v ...any) error {
	_, err := fmt.Fprintf(gc.stdout, format, v...)
	if err != nil {
		return fmt.Errorf("writing to stdout: %w", err)
	}

	return nil
}
