package runner

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// OCIStepResource knows how to load a step from an OCI image (artifact)
type OCIStepResource struct {
	fetcher    *oci.OCIFetcher
	registry   string
	repository string
	stepDir    string
	tag        string
	filename   string
}

func NewOCIStepResource(fetcher *oci.OCIFetcher, registry, repository string, tag string, stepDir string, filename string) *OCIStepResource {
	return &OCIStepResource{
		fetcher:    fetcher,
		registry:   registry,
		repository: repository,
		tag:        tag,
		stepDir:    stepDir,
		filename:   filename,
	}
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

func (sr *OCIStepResource) Fetch(ctx context.Context, _ *expression.InterpolationContext) (*proto.SpecDefinition, error) {
	imgRef, err := sr.NamedReference()
	if err != nil {
		return nil, fmt.Errorf("fetching oci step: %w", err)
	}

	dir, err := sr.fetcher.Fetch(ctx, imgRef)
	if err != nil {
		return nil, fmt.Errorf("fetching oci step: %w", err)
	}

	specDef, err := NewFileSystemStepResource(filepath.Join(dir, sr.stepDir), sr.filename).Fetch(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching oci step: %w", err)
	}

	return specDef, nil
}
