package runner

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
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

func (sr *GitStepResource) Fetch(ctx context.Context, view *expression.InterpolationContext) (*SpecDefinition, error) {
	url, err := expression.ExpandString(view, sr.url)
	if err != nil {
		return nil, fmt.Errorf("fetching git step: interpolating url: %w", err)
	}

	dir, err := sr.fetcher.Get(ctx, url, sr.version)
	if err != nil {
		return nil, fmt.Errorf("fetching git step: %w", err)
	}

	specDef, err := NewFileSystemStepResource(dir, sr.stepDir, sr.filename).Fetch(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching git step: %w", err)
	}

	return specDef, nil
}
