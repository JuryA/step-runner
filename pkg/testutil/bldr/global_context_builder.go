package bldr

import (
	"bytes"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type GlobalContextBuilder struct {
	job map[string]string
}

func GlobalContext() *GlobalContextBuilder {
	return &GlobalContextBuilder{
		job: map[string]string{},
	}
}

func (bldr *GlobalContextBuilder) WithJob(name, value string) *GlobalContextBuilder {
	bldr.job[name] = value
	return bldr
}

func (bldr *GlobalContextBuilder) Build() *runner.GlobalContext {
	return &runner.GlobalContext{
		WorkDir: ".",
		Job:     bldr.job,
		Env:     runner.NewEmptyEnvironment(),
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}
