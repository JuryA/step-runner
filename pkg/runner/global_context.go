package runner

import (
	"io"
	"os"
)

type GlobalContext struct {
	WorkDir string
	Job     map[string]string
	Env     *Environment
	Stdout  io.Writer
	Stderr  io.Writer
}

func NewGlobalContext(env *Environment) *GlobalContext {
	return &GlobalContext{
		Job:    map[string]string{},
		Env:    env,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}
