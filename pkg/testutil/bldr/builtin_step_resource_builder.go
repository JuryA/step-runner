package bldr

import (
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type BuiltInStepResourceBuilder struct {
	path []string
}

func BuiltInStepResource() *BuiltInStepResourceBuilder {
	return &BuiltInStepResourceBuilder{
		path: []string{"oci", "publish"},
	}
}

func (b *BuiltInStepResourceBuilder) WithStep(step string) *BuiltInStepResourceBuilder {
	b.path = strings.Split(step, "/")
	return b
}

func (b *BuiltInStepResourceBuilder) Build() *runner.BuiltInStepResource {
	return runner.NewBuiltInStepResource(b.path, "step.yml")
}
