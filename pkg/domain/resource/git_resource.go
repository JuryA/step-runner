package resource

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
)

type GitResource struct {
	gitFetcher *git.GitFetcher
	url        string
	version    string
	path       []string
	filename   string
}

func NewGitResource(gitFetcher *git.GitFetcher, url, version string, path []string, filename string) *GitResource {
	return &GitResource{
		gitFetcher: gitFetcher,
		url:        url,
		version:    version,
		path:       path,
		filename:   filename,
	}
}

func (l *GitResource) Load(ctx context.Context) (string, string, error) {
	clonedDir, err := l.gitFetcher.Get(ctx, l.url, l.version)

	if err != nil {
		return "", "", fmt.Errorf("failed to load git resource %s@%s: %w", l.url, l.version, err)
	}

	contents, filenameDir, err := NewFileResource(clonedDir, l.path, l.filename).Load(ctx)

	if err != nil {
		return "", "", fmt.Errorf("failed to load git resource %s@%s: %w", l.url, l.version, err)
	}

	return contents, filenameDir, nil
}
