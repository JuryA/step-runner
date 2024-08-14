package schema

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Parser struct {
	globalCtx *runner.GlobalContext
	stepCache runner.Cache
}

func NewParser(globalCtx *runner.GlobalContext, stepCache runner.Cache) *Parser {
	return &Parser{
		globalCtx: globalCtx,
		stepCache: stepCache,
	}
}

func (p *Parser) Parse(specDef *proto.SpecDefinition, params *runner.Params) (runner.Step, error) {
	if err := p.validateInputs(specDef.Spec, params.Inputs); err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	step, err := p.parseStepType(specDef)

	if err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	return step, nil
}

func (p *Parser) parseStepType(specDef *proto.SpecDefinition) (runner.Step, error) {
	if specDef.Definition.Type == proto.DefinitionType_exec {
		return runner.NewExecutableStep(), nil
	}

	if specDef.Definition.Type == proto.DefinitionType_steps {
		return runner.NewSequenceOfSteps(p.globalCtx, p.stepCache, p), nil
	}

	return nil, fmt.Errorf("unknown step definition type: %s", specDef.Definition.Type)
}

func (p *Parser) validateInputs(spec *proto.Spec, inputs map[string]*context.Variable) error {
	for key, value := range spec.Spec.Inputs {
		if inputs[key] == nil && value.Default == nil {
			return fmt.Errorf("input %q required, but not defined", key)
		}
	}

	for key := range inputs {
		if spec.Spec.Inputs[key] == nil {
			return fmt.Errorf("input %q not found", key)
		}
	}

	return nil
}
