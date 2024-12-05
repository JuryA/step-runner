package runner

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type InlineStepResource struct {
	specDef *proto.SpecDefinition
}

func NewInlineStepResource(specDef *proto.SpecDefinition) *InlineStepResource {
	return &InlineStepResource{
		specDef: specDef,
	}
}

func (sr *InlineStepResource) ToProtoStepRef() *proto.Step_Reference {
	return nil
}

func (sr *InlineStepResource) Interpolate(_ *expression.InterpolationContext) (StepResource, error) {
	return sr, nil
}

func (sr *InlineStepResource) Describe() string {
	return fmt.Sprintf("spec def")
}

func (sr *InlineStepResource) ToSpecDef() *proto.SpecDefinition {
	return sr.specDef
}
