package bldr

import (
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type EnvironmentBuilder struct {
	env map[string]string
}

func Env() *EnvironmentBuilder {
	return &EnvironmentBuilder{
		env: make(map[string]string),
	}
}

func (bldr *EnvironmentBuilder) Build() *runner.Environment {
	return runner.NewEnvironment(bldr.env)
}
