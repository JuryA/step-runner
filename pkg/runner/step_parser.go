package runner

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepParser interface {
	Parse(*proto.SpecDefinition, *Params) (Step, error)
}

type Parser struct {
	globalCtx *GlobalContext
	stepCache Cache
}

func NewParser(globalCtx *GlobalContext, stepCache Cache) *Parser {
	return &Parser{
		globalCtx: globalCtx,
		stepCache: stepCache,
	}
}

func (p *Parser) Parse(specDef *proto.SpecDefinition, params *Params) (Step, error) {
	if err := p.validateInputs(specDef.Spec, params.Inputs); err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	step, err := p.parseStepType(specDef)

	if err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	return step, nil
}

func (p *Parser) parseStepType(specDef *proto.SpecDefinition) (Step, error) {
	if specDef.Definition.Type == proto.DefinitionType_exec {
		return NewExecutableStep(specDef), nil
	}

	if specDef.Definition.Type == proto.DefinitionType_steps {
		var steps []Step

		for _, stepReference := range specDef.Definition.Steps {
			steps = append(steps, NewLazilyLoadedStep(p.globalCtx, p.stepCache, p, stepReference))
		}

		return NewSequenceOfSteps(steps...), nil
	}

	return nil, fmt.Errorf("unknown step definition type: %s", specDef.Definition.Type)
}

func (p *Parser) validateInputs(spec *proto.Spec, inputs map[string]*context.Variable) error {
	for key, value := range spec.Spec.Inputs {
		if inputs[key] == nil && value.Default == nil {
			return fmt.Errorf("input %q required, but not defined", key)
		}
	}

	return nil
}
