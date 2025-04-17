package runner

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
)

// DistStepResource knows how to load a step that is internal to the step-runner
type DistStepResource struct {
	fetcher  *dist.Fetcher
	stepDir  string
	filename string
}

func NewDistStepResource(fetcher *dist.Fetcher, stepDir string, filename string) *DistStepResource {
	return &DistStepResource{
		fetcher:  fetcher,
		stepDir:  stepDir,
		filename: filename,
	}
}

func (sr *DistStepResource) Fetch(ctx context.Context, _ *expression.InterpolationContext) (*SpecDefinition, error) {
	dir, err := sr.fetcher.Fetch(sr.stepDir)
	if err != nil {
		return nil, fmt.Errorf("fetching dist step: %w", err)
	}

	specDef, err := NewFileSystemStepResource(dir, sr.stepDir, sr.filename).Fetch(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching dist step: %w", err)
	}

	return specDef, nil
}
