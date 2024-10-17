package runner

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// GitStepResource knows how to load a step from a Git repository
type GitStepResource struct {
	url      string
	path     []string
	version  string
	filename string
}

func NewGitStepResource(url string, version string, path []string, filename string) *GitStepResource {
	return &GitStepResource{
		url:      url,
		version:  version,
		path:     path,
		filename: filename,
	}
}

func (sr *GitStepResource) Interpolate(view *expression.InterpolationContext) (StepResource, error) {
	newURL, err := expression.ExpandString(view, sr.url)

	if err != nil {
		return nil, fmt.Errorf("failed to interpolate git url: %w", err)
	}

	return NewGitStepResource(newURL, sr.version, sr.path, sr.filename), nil
}

func (sr *GitStepResource) ToProtoStepRef() *proto.Step_Reference {
	return &proto.Step_Reference{
		Url:      sr.url,
		Protocol: proto.StepReferenceProtocol_git,
		Path:     sr.path,
		Filename: sr.filename,
		Version:  sr.version,
	}
}

func (sr *GitStepResource) Describe() string {
	return fmt.Sprintf("%s@%s:%s/%s", sr.url, sr.version, strings.Join(sr.path, "/"), sr.filename)
}
