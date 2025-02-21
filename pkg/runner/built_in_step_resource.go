package runner

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// BuiltInStepResource knows how to load a step that is internal to the step-runner
type BuiltInStepResource struct {
	path     []string
	filename string
}

func NewBuiltInStepResource(path []string, filename string) *BuiltInStepResource {
	return &BuiltInStepResource{
		path:     path,
		filename: filename,
	}
}

func (sr *BuiltInStepResource) Interpolate(_ *expression.InterpolationContext) (StepResource, error) {
	return sr, nil
}

func (sr *BuiltInStepResource) ToProtoStepRef() *proto.Step_Reference {
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_builtin,
		Path:     sr.path,
		Filename: sr.filename,
	}
}

func (sr *BuiltInStepResource) Describe() string {
	return fmt.Sprintf("builtin:%s/%s", strings.Join(sr.path, "/"), sr.filename)
}
