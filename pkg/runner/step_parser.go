package runner

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepParser interface {
	Parse(specDef *proto.SpecDefinition, params *Params, loadedFrom StepReference) (Step, error)
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

func (p *Parser) Parse(specDef *proto.SpecDefinition, params *Params, loadedFrom StepReference) (Step, error) {
	if err := p.validateInputs(specDef.Spec, params.Inputs); err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	step, err := p.parseStepType(specDef, params, loadedFrom)

	if err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	return step, nil
}

func (p *Parser) parseStepType(specDef *proto.SpecDefinition, params *Params, loadedFrom StepReference) (Step, error) {
	if specDef.Definition.Type == proto.DefinitionType_exec {
		return NewExecutableStep(loadedFrom, params, specDef), nil
	}

	if specDef.Definition.Type == proto.DefinitionType_steps {
		var steps []Step

		for _, stepReference := range specDef.Definition.Steps {
			stepResource, err := p.parseStepResource(stepReference.Step)

			if err != nil {
				return nil, err
			}

			steps = append(steps, NewLazilyLoadedStep(p.globalCtx, p.stepCache, p, stepReference, stepResource, specDef.Dir))
		}

		return NewSequenceOfSteps(loadedFrom, params, specDef, steps...), nil
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

func (p *Parser) parseStepResource(stepRef *proto.Step_Reference) (StepResource, error) {
	switch stepRef.Protocol {
	case proto.StepReferenceProtocol_local:
		return NewFileSystemStepResource(stepRef.Path, stepRef.Filename), nil

	case proto.StepReferenceProtocol_git:
		return NewGitStepResource(stepRef.Url, stepRef.Version, stepRef.Path, stepRef.Filename), nil

	case proto.StepReferenceProtocol_oci:
		return NewOCIStepResource(stepRef.Registry, stepRef.Repository, stepRef.Tag, stepRef.Path, stepRef.Filename), nil

	case proto.StepReferenceProtocol_dist:
		return NewDistStepResource(stepRef.Path, stepRef.Filename), nil
	}

	return nil, fmt.Errorf("unknown step reference protocol: %s", stepRef.Protocol)
}
