package runner

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// OCIStepResource knows how to load a step from an OCI image (artifact)
type OCIStepResource struct {
	url      string
	path     []string
	tag      string
	filename string
}

func NewOCIStepResource(url string, tag string, path []string, filename string) *OCIStepResource {
	return &OCIStepResource{
		url:      url,
		tag:      tag,
		path:     path,
		filename: filename,
	}
}

func (sr *OCIStepResource) Interpolate(_ *expression.InterpolationContext) (StepResource, error) {
	return sr, nil
}

func (sr *OCIStepResource) ToProtoStepRef() *proto.Step_Reference {
	return &proto.Step_Reference{
		Url:      sr.url,
		Protocol: proto.StepReferenceProtocol_oci,
		Path:     sr.path,
		Filename: sr.filename,
		Version:  sr.tag,
	}
}

func (sr *OCIStepResource) Describe() string {
	return fmt.Sprintf("%s:%s %s/%s", sr.url, sr.tag, strings.Join(sr.path, "/"), sr.filename)
}
