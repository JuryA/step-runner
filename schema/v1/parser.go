package schema

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Parser struct {
	stepCache runner.Cache
	runStep   runner.LegacyRunStepFn
}

func NewParser(stepCache runner.Cache, runStep runner.LegacyRunStepFn) *Parser {
	return &Parser{
		stepCache: stepCache,
		runStep:   runStep,
	}
}

func (p *Parser) Parse(specDef *proto.SpecDefinition) (runner.Step, error) {
	if specDef.Definition.Type == proto.DefinitionType_exec {
		return runner.NewExecutableStep(), nil
	}

	if specDef.Definition.Type == proto.DefinitionType_steps {
		return runner.NewSequenceOfSteps(p.stepCache, p, p.runStep), nil
	}

	return nil, fmt.Errorf("unknown step definition type: %s", specDef.Definition.Type)
}
