package runner

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// FileSystemStepResource knows how to load a step from the file system
type FileSystemStepResource struct {
	path     []string
	filename string
}

func NewFileSystemStepResource(path []string, filename string) *FileSystemStepResource {
	return &FileSystemStepResource{
		path:     path,
		filename: filename,
	}
}

func (sr *FileSystemStepResource) ToProtoStepRef() *proto.Step_Reference {
	return &proto.Step_Reference{
		Url:      "",
		Protocol: proto.StepReferenceProtocol_local,
		Path:     sr.path,
		Filename: sr.filename,
		Version:  "",
	}
}

func (sr *FileSystemStepResource) Interpolate(_ *expression.InterpolationContext) (StepResource, error) {
	return sr, nil
}

func (sr *FileSystemStepResource) Describe() string {
	return fmt.Sprintf("%s/%s", strings.Join(sr.path, "/"), sr.filename)
}
