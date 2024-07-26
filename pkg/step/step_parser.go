package step

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/domain"
	"gitlab.com/gitlab-org/step-runner/pkg/domain/resource"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

type StepParser struct {
	gitFetcher  *git.GitFetcher
	stepFactory StepFactory
}

func NewStepParser(stepFactory StepFactory, gitFetcher *git.GitFetcher) *StepParser {
	return &StepParser{
		gitFetcher:  gitFetcher,
		stepFactory: stepFactory,
	}
}

func (p *StepParser) Parse(rawSteps string) (domain.Step, error) {
	stepDef, err := wrapStepsInSpecDef(rawSteps)

	if err != nil {
		return nil, fmt.Errorf("failed to parse steps: %w", err)
	}

	protoDef, err := CompileSteps(stepDef)

	if err != nil {
		return nil, fmt.Errorf("failed to parse steps: %w", err)
	}

	step, err := p.compileToDomainSteps(stepDef, protoDef)

	if err != nil {
		return nil, fmt.Errorf("failed to parse steps: %w", err)
	}

	return step, nil
}

func wrapStepsInSpecDef(steps string) (*schema.StepDefinition, error) {
	specDef := &schema.StepDefinition{
		Spec:       &schema.Spec{},
		Definition: &schema.Definition{},
	}
	err := yaml.Unmarshal([]byte(steps), &specDef.Definition.Steps)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling steps: %w", err)
	}
	runningSteps, _ := yaml.Marshal(specDef)
	fmt.Printf("running steps:\n%v", string(runningSteps))
	return specDef, nil
}

func (p *StepParser) compileToDomainSteps(stepDef *schema.StepDefinition, protoDef *proto.SpecDefinition) (domain.Step, error) {
	inputs := p.buildInputs(stepDef)
	outputs := p.buildOutputs(stepDef)

	if len(protoDef.Definition.Steps) > 0 {
		return p.buildMultiStep(inputs, outputs, stepDef, protoDef)
	}

	return nil, nil
}

func (p *StepParser) buildInputs(stepDef *schema.StepDefinition) *domain.Inputs {
	inputs := make([]*domain.Input, 0)

	for name, values := range stepDef.Spec.Spec.Inputs {
		inputs = append(inputs, domain.NewInput(name, values.Type, values.Default, values.Sensitive)) // do something more sensible with Type here
	}

	return domain.NewInputs(inputs...)
}

func (p *StepParser) buildOutputs(stepDef *schema.StepDefinition) *domain.Outputs {
	outputs := make([]*domain.Output, 0)

	for name, values := range stepDef.Spec.Spec.Outputs.Outputs {
		outputs = append(outputs, domain.NewOutput(name, values.Type, values.Default, values.Sensitive)) // do something more sensible with Type here
	}

	return domain.NewOutputs(stepDef.Spec.Spec.Outputs.Delegate, outputs...)
}

func (p *StepParser) buildMultiStep(inputs *domain.Inputs, outputs *domain.Outputs, stepDef *schema.StepDefinition, specDef *proto.SpecDefinition) (*domain.MultiStep, error) {
	steps := make([]domain.Step, len(stepDef.Definition.Steps))

	for i, subStepDef := range specDef.Definition.Steps {
		// sensible validation in here

		switch subStepDef.Step.Protocol {
		case proto.StepReferenceProtocol_local:
			loader := resource.NewFileResource(stepDef.Dir, subStepDef.Step.Path, subStepDef.Step.Filename)
			steps[i] = p.stepFactory.CreateLazilyLoadedStep(p, subStepDef.Name, loader)
		case proto.StepReferenceProtocol_git:
			loader := resource.NewGitResource(p.gitFetcher, subStepDef.Step.Url, subStepDef.Step.Version, subStepDef.Step.Path, subStepDef.Step.Filename)
			steps[i] = p.stepFactory.CreateLazilyLoadedStep(p, subStepDef.Name, loader)
		}

	}

	return domain.NewMultiStep(steps...), nil
}
