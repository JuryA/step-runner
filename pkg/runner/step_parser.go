package runner

import (
	"fmt"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepParser interface {
	Parse(globalCtx *GlobalContext, specDef *proto.SpecDefinition, params *Params, loadedFrom StepReference) (Step, error)
}

type Parser struct {
	stepCache  Cache
	gitFetcher *git.GitFetcher
	ociFetcher *oci.OCIFetcher
}

func NewParser(stepCache Cache, gitFetcher *git.GitFetcher, ociFetcher *oci.OCIFetcher) *Parser {
	return &Parser{
		stepCache:  stepCache,
		gitFetcher: gitFetcher,
		ociFetcher: ociFetcher,
	}
}

func (p *Parser) Parse(globalCtx *GlobalContext, specDef *proto.SpecDefinition, params *Params, loadedFrom StepReference) (Step, error) {
	if err := p.validateInputs(specDef.Spec, params.Inputs); err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	step, err := p.parseStepType(globalCtx, specDef, params, loadedFrom)

	if err != nil {
		return nil, fmt.Errorf("failed to parse spec definition: %w", err)
	}

	return step, nil
}

func (p *Parser) parseStepType(globalCtx *GlobalContext, specDef *proto.SpecDefinition, params *Params, loadedFrom StepReference) (Step, error) {
	if specDef.Definition.Type == proto.DefinitionType_exec {
		return NewExecutableStep(loadedFrom, params, specDef), nil
	}

	if specDef.Definition.Type == proto.DefinitionType_steps {
		var steps []Step

		for _, stepReference := range specDef.Definition.Steps {
			stepResource, err := p.parseStepResource(specDef.Dir, stepReference.Step)

			if err != nil {
				return nil, err
			}

			steps = append(steps, NewLazilyLoadedStep(globalCtx, p.stepCache, p, stepReference, stepResource))
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

func (p *Parser) parseStepResource(parentDir string, stepRef *proto.Step_Reference) (StepResource, error) {
	stepDir := filepath.Join(stepRef.Path...)

	switch stepRef.Protocol {
	case proto.StepReferenceProtocol_local:
		return NewFileSystemStepResource(filepath.Join(parentDir, stepDir), stepRef.Filename), nil

	case proto.StepReferenceProtocol_git:
		return NewGitStepResource(p.gitFetcher, stepRef.Url, stepRef.Version, stepDir, stepRef.Filename), nil

	case proto.StepReferenceProtocol_oci:
		return NewOCIStepResource(p.ociFetcher, stepRef.Registry, stepRef.Repository, stepRef.Tag, stepDir, stepRef.Filename), nil

	case proto.StepReferenceProtocol_dist:
		return NewDistStepResource(stepRef.Path, stepRef.Filename), nil
	}

	return nil, fmt.Errorf("unknown step reference protocol: %s", stepRef.Protocol)
}
