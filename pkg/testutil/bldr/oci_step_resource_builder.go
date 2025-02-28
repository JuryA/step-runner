package bldr

import (
	"github.com/google/go-containerregistry/pkg/name"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type OCIStepResourceBuilder struct {
	registry   string
	repository string
	tag        string
	path       []string
}

func OCIStepResource() *OCIStepResourceBuilder {
	return &OCIStepResourceBuilder{
		registry:   "registry.gitlab.com",
		repository: "gitlab-org/step-runner",
		tag:        "latest",
		path:       []string{},
	}
}

func (b *OCIStepResourceBuilder) WithImgRef(imgRef name.Reference) *OCIStepResourceBuilder {
	b.registry = imgRef.Context().RegistryStr()
	b.repository = imgRef.Context().RepositoryStr()
	b.tag = imgRef.Identifier()
	return b
}

func (b *OCIStepResourceBuilder) WithPath(path ...string) *OCIStepResourceBuilder {
	b.path = path
	return b
}

func (b *OCIStepResourceBuilder) Build() *runner.OCIStepResource {
	return runner.NewOCIStepResource(b.registry, b.repository, b.tag, b.path, "step.yml")
}
