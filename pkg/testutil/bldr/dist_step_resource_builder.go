package bldr

import (
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type DistStepResourceBuilder struct {
	path []string
}

func DistStepResource() *DistStepResourceBuilder {
	return &DistStepResourceBuilder{
		path: []string{"oci", "publish"},
	}
}

func (b *DistStepResourceBuilder) WithStep(step string) *DistStepResourceBuilder {
	b.path = strings.Split(step, "/")
	return b
}

func (b *DistStepResourceBuilder) Build() *runner.DistStepResource {
	return runner.NewDistStepResource(b.path, "step.yml")
}
