package runner

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepParser interface {
	Parse(globalCtx *GlobalContext, specDef *SpecDefinition, params *Params, loadedFrom StepReference) (Step, error)
}

type Parser struct {
	stepResParser *StepResourceParser
}

func NewParser(stepResParser *StepResourceParser) *Parser {
	return &Parser{
		stepResParser: stepResParser,
	}
}

func (p *Parser) Parse(globalCtx *GlobalContext, specDef *SpecDefinition, params *Params, loadedFrom StepReference) (Step, error) {
	if err := p.validateInputs(specDef.SpecInputs(), params.Inputs); err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	step, err := p.parseStepType(globalCtx, specDef, params, loadedFrom)

	if err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	return step, nil
}

func (p *Parser) parseStepType(globalCtx *GlobalContext, specDef *SpecDefinition, params *Params, loadedFrom StepReference) (Step, error) {
	if specDef.IsTypeExec() {
		return NewExecutableStep(loadedFrom, params, specDef), nil
	}

	if specDef.IsTypeSteps() {
		var steps []Step

		for _, stepReference := range specDef.Steps() {
			stepResource, err := p.stepResParser.Parse(specDef.Dir(), stepReference.Step)

			if err != nil {
				return nil, err
			}

			steps = append(steps, NewLazilyLoadedStep(globalCtx, p, stepReference, stepResource))
		}

		return NewSequenceOfSteps(loadedFrom, params, specDef, steps...), nil
	}

	return nil, fmt.Errorf("unknown step definition type: %s", specDef.DescribeType())
}

func (p *Parser) validateInputs(specInputs map[string]*proto.Spec_Content_Input, inputs map[string]*context.Variable) error {
	for key, value := range specInputs {
		if inputs[key] == nil && value.Default == nil {
			return fmt.Errorf("input %q required, but not defined", key)
		}
	}

	return nil
}
