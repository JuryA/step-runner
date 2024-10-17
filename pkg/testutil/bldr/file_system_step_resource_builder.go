package bldr

import (
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type FileSystemStepResourceBuilder struct {
}

func FileSystemStepResource() *FileSystemStepResourceBuilder {
	return &FileSystemStepResourceBuilder{}
}

func (bldr *FileSystemStepResourceBuilder) Build() *runner.FileSystemStepResource {
	return runner.NewFileSystemStepResource([]string{}, "step.yml")
}
