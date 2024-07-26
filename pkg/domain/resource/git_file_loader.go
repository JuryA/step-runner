package resource

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
)

type GitFileLoader struct {
	gitFetcher *git.GitFetcher
	url        string
	version    string
	path       []string
	filename   string
}

func NewGitFileLoader(gitFetcher *git.GitFetcher, url, version string, path []string, filename string) *GitFileLoader {
	return &GitFileLoader{
		gitFetcher: gitFetcher,
		url:        url,
		version:    version,
		path:       path,
		filename:   filename,
	}
}

func (l *GitFileLoader) Load(ctx context.Context) ([]byte, error) {
	dir, err := l.gitFetcher.Get(ctx, l.url, l.version)

	if err != nil {
		return nil, fmt.Errorf("failed to load git resource %s@%s: %w", l.url, l.version, err)
	}

	contents, err := NewLocalFileLoader(dir, l.path, l.filename).Load(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to load git resource %s@%s: %w", l.url, l.version, err)
	}

	return contents, nil
}
