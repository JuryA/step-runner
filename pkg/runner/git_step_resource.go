package runner

import (
	"context"
	"fmt"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// GitStepResource knows how to load a step from a Git repository
type GitStepResource struct {
	fetcher  *git.GitFetcher
	url      string
	stepDir  string
	version  string
	filename string
}

func NewGitStepResource(fetcher *git.GitFetcher, url string, version string, stepDir string, filename string) *GitStepResource {
	return &GitStepResource{
		fetcher:  fetcher,
		url:      url,
		version:  version,
		stepDir:  stepDir,
		filename: filename,
	}
}

func (sr *GitStepResource) Interpolate(view *expression.InterpolationContext) (StepResource, error) {
	newURL, err := expression.ExpandString(view, sr.url)

	if err != nil {
		return nil, fmt.Errorf("failed to interpolate git url: %w", err)
	}

	return NewGitStepResource(sr.fetcher, newURL, sr.version, sr.stepDir, sr.filename), nil
}

func (sr *GitStepResource) Fetch(ctx context.Context) (*proto.SpecDefinition, error) {
	dir, err := sr.fetcher.Get(ctx, sr.url, sr.version)
	if err != nil {
		return nil, fmt.Errorf("fetching git step: %w", err)
	}

	specDef, err := NewFileSystemStepResource(filepath.Join(dir, sr.stepDir), sr.filename).Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching git step: %w", err)
	}

	return specDef, nil
}

func (sr *GitStepResource) Describe() string {
	return fmt.Sprintf("%s@%s:%s/%s", sr.url, sr.version, sr.stepDir, sr.filename)
}
