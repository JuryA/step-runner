package runner

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// OCIStepResource knows how to load a step from an OCI regsitry
type OCIStepResource struct {
	url      string
	path     []string
	version  string
	filename string
}

func NewOCIStepResource(url string, version string, path []string, filename string) *OCIStepResource {
	return &OCIStepResource{
		url:      url,
		version:  version,
		path:     path,
		filename: filename,
	}
}

func (sr *OCIStepResource) Interpolate(view *expression.InterpolationContext) (StepResource, error) {
	newURL, err := expression.ExpandString(view, sr.url)

	if err != nil {
		return nil, fmt.Errorf("failed to interpolate oci url: %w", err)
	}

	return NewOCIStepResource(newURL, sr.version, sr.path, sr.filename), nil
}

func (sr *OCIStepResource) ToProtoStepRef() *proto.Step_Reference {
	return &proto.Step_Reference{
		Url:      sr.url,
		Protocol: proto.StepReferenceProtocol_oci,
		Path:     sr.path,
		Filename: sr.filename,
		Version:  sr.version,
	}
}

func (sr *OCIStepResource) Describe() string {
	return fmt.Sprintf("%s@%s:%s/%s", sr.url, sr.version, strings.Join(sr.path, "/"), sr.filename)
}
