package bldr

import (
	"path/filepath"
	"testing"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type GitStepResourceBuilder struct {
	t       *testing.T
	url     string
	version string
	path    string
}

func GitStepResource(t *testing.T) *GitStepResourceBuilder {
	return &GitStepResourceBuilder{
		t:       t,
		url:     "https://gitlab.com/steps/echo",
		version: "main",
		path:    "",
	}
}

func (bldr *GitStepResourceBuilder) WithURL(url string) *GitStepResourceBuilder {
	bldr.url = url
	return bldr
}

func (bldr *GitStepResourceBuilder) WithVersion(version string) *GitStepResourceBuilder {
	bldr.version = version
	return bldr
}

func (bldr *GitStepResourceBuilder) WithPath(path ...string) *GitStepResourceBuilder {
	bldr.path = filepath.Join(path...)
	return bldr
}

func (bldr *GitStepResourceBuilder) Build() *runner.GitStepResource {
	gitFetcher := git.New(bldr.t.TempDir(), git.CloneOptions{Depth: 0})
	return runner.NewGitStepResource(gitFetcher, bldr.url, bldr.version, bldr.path, "step.yml")
}
