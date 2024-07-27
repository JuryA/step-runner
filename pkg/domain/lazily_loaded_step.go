package domain

import (
	goctx "context"
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/domain/resource"
)

type LazilyLoadedStep struct {
	parser   StepParser
	name     string
	resource resource.Resource
}

func NewLazilyLoadedStep(parser StepParser, name string, resource resource.Resource) *LazilyLoadedStep {
	return &LazilyLoadedStep{
		parser:   parser,
		name:     name,
		resource: resource,
	}
}

func (lls *LazilyLoadedStep) Run(ctx goctx.Context, globalCtx *GlobalCtx, stepCtx *StepsCtx) (StepResult, error) {
	yamlSteps, dir, err := lls.resource.Load(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to run lazily loaded step %q: %w", lls.name, err)
	}

	step, _, err := lls.parser.Parse(yamlSteps, dir)

	if err != nil {
		return nil, fmt.Errorf("failed to run lazily loaded step %q: %w", lls.name, err)
	}

	return step.Run(ctx, globalCtx, stepCtx)
}
