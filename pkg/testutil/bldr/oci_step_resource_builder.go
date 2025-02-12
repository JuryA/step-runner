package bldr

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type OCIStepResourceBuilder struct {
	url  string
	tag  string
	path []string
}

func OCIStepResource() *OCIStepResourceBuilder {
	return &OCIStepResourceBuilder{
		url:  "registry.gitlab.com/gitlab-org/step-runner",
		tag:  "latest",
		path: []string{},
	}
}

func (b *OCIStepResourceBuilder) WithImgRef(imgRef name.Reference) *OCIStepResourceBuilder {
	b.tag = imgRef.Identifier()
	b.url = strings.TrimSuffix(imgRef.Name(), ":"+b.tag)
	return b
}

func (b *OCIStepResourceBuilder) Build() *runner.OCIStepResource {
	return runner.NewOCIStepResource(b.url, b.tag, b.path, "step.yml")
}
