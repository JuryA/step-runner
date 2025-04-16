package runner

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

type stepResourceParser interface {
	Parse(parentDir string, stepRef *proto.Step_Reference) (StepResource, error)
}

// DynamicStepResource knows how to convert an expression into a step resource
type DynamicStepResource struct {
	shortRef string
	parser   stepResourceParser
}

func NewDynamicStepResource(parser stepResourceParser, shortRef string) *DynamicStepResource {
	return &DynamicStepResource{
		shortRef: shortRef,
		parser:   parser,
	}
}

func (sr *DynamicStepResource) Fetch(ctx context.Context, view *expression.InterpolationContext) (*SpecDefinition, error) {
	shortRef, err := expression.ExpandString(view, sr.shortRef)
	if err != nil {
		return nil, fmt.Errorf("fetching step: interpolating reference: %w", err)
	}

	stepRef, err := schema.CompileShortRef(shortRef)
	if err != nil {
		return nil, fmt.Errorf("fetching step: compiling step reference: %w", err)
	}

	stepRsc, err := sr.parser.Parse("", stepRef)
	if err != nil {
		return nil, fmt.Errorf("fetching step: %w", err)
	}

	return stepRsc.Fetch(ctx, view)
}
