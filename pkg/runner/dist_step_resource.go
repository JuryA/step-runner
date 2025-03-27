package runner

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// DistStepResource knows how to load a step that is internal to the step-runner
type DistStepResource struct {
	path     []string
	filename string
}

func NewDistStepResource(path []string, filename string) *DistStepResource {
	return &DistStepResource{
		path:     path,
		filename: filename,
	}
}

func (sr *DistStepResource) Interpolate(_ *expression.InterpolationContext) (StepResource, error) {
	return sr, nil
}

func (sr *DistStepResource) ToProtoStepRef() *proto.Step_Reference {
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_dist,
		Path:     sr.path,
		Filename: sr.filename,
	}
}

func (sr *DistStepResource) Describe() string {
	return fmt.Sprintf("dist:%s/%s", strings.Join(sr.path, "/"), sr.filename)
}
