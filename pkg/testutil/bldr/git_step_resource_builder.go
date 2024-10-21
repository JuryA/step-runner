package bldr

import (
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type GitStepResourceBuilder struct {
	url     string
	version string
	path    []string
}

func GitStepResource() *GitStepResourceBuilder {
	return &GitStepResourceBuilder{
		url:     "https://gitlab.com/steps/echo",
		version: "main",
		path:    []string{""},
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
	bldr.path = path
	return bldr
}

func (bldr *GitStepResourceBuilder) Build() *runner.GitStepResource {
	return runner.NewGitStepResource(bldr.url, bldr.version, bldr.path, "step.yml")
}
