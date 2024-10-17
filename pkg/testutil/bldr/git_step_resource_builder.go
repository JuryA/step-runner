package bldr

import (
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type GitStepResourceBuilder struct {
	url string
}

func GitStepResource() *GitStepResourceBuilder {
	return &GitStepResourceBuilder{
		url: "https://gitlab.com/steps/echo",
	}
}

func (bldr *GitStepResourceBuilder) WithURL(url string) *GitStepResourceBuilder {
	bldr.url = url
	return bldr
}

func (bldr *GitStepResourceBuilder) Build() *runner.GitStepResource {
	return runner.NewGitStepResource(
		bldr.url,
		"main",
		[]string{},
		"step.yml",
	)
}
