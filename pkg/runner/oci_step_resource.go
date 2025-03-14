package runner

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// OCIStepResource knows how to load a step from an OCI image (artifact)
type OCIStepResource struct {
	registry   string
	repository string
	path       []string
	tag        string
	filename   string
}

func NewOCIStepResource(registry, repository string, tag string, path []string, filename string) *OCIStepResource {
	return &OCIStepResource{
		registry:   registry,
		repository: repository,
		tag:        tag,
		path:       path,
		filename:   filename,
	}
}

func (sr *OCIStepResource) Interpolate(_ *expression.InterpolationContext) (StepResource, error) {
	return sr, nil
}

func (sr *OCIStepResource) ToProtoStepRef() *proto.Step_Reference {
	return &proto.Step_Reference{
		Protocol:   proto.StepReferenceProtocol_oci,
		Registry:   sr.registry,
		Repository: sr.repository,
		Tag:        sr.tag,
		Path:       sr.path,
		Filename:   sr.filename,
	}
}

func (sr *OCIStepResource) Describe() string {
	return fmt.Sprintf("%s/%s:%s[%s/%s]", sr.registry, sr.repository, sr.tag, strings.Join(sr.path, "/"), sr.filename)
}

func (sr *OCIStepResource) NamedReference() (name.Reference, error) {
	repository := path.Join(sr.registry, sr.repository)

	imgRefTag, tagErr := name.ParseReference(fmt.Sprintf("%s:%s", repository, sr.tag))
	if tagErr == nil {
		return imgRefTag, nil
	}

	digestImgRef, digestErr := name.ParseReference(fmt.Sprintf("%s@%s", repository, sr.tag))
	if digestErr == nil {
		return digestImgRef, nil
	}

	return nil, fmt.Errorf("parsing OCI image reference: %w", tagErr)
}
