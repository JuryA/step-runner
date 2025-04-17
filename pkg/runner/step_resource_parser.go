package runner

import (
	"fmt"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepResourceParser struct {
	gitFetcher  *git.GitFetcher
	distFetcher *dist.Fetcher
}

func NewStepResourceParser(gitFetcher *git.GitFetcher, distFetcher *dist.Fetcher) *StepResourceParser {
	return &StepResourceParser{
		gitFetcher:  gitFetcher,
		distFetcher: distFetcher,
	}
}

func (p *StepResourceParser) Parse(workDir string, stepRef *proto.Step_Reference) (StepResource, error) {
	stepPath := p.stepPath(stepRef)

	switch stepRef.Protocol {
	case proto.StepReferenceProtocol_local:
		return NewFileSystemStepResource(workDir, stepPath, stepRef.Filename), nil

	case proto.StepReferenceProtocol_git:
		return NewGitStepResource(p.gitFetcher, stepRef.Url, stepRef.Version, stepPath, stepRef.Filename), nil

	case proto.StepReferenceProtocol_dist:
		return NewDistStepResource(p.distFetcher, stepPath, stepRef.Filename), nil

	case proto.StepReferenceProtocol_dynamic:
		return NewDynamicStepResource(p, stepRef.Url), nil

	case proto.StepReferenceProtocol_spec_def:
		return NewFixedStepResource(NewSpecDefinition(stepRef.SpecDef.Spec, stepRef.SpecDef.Definition, stepPath)), nil
	}

	return nil, fmt.Errorf("unknown step reference protocol: %s", stepRef.Protocol)
}

func (p *StepResourceParser) stepPath(stepRef *proto.Step_Reference) string {
	if val, ok := stepRef.StepPath.(*proto.Step_Reference_PathExp); ok {
		return val.PathExp
	}

	if val, ok := stepRef.StepPath.(*proto.Step_Reference_Paths); ok {
		return filepath.Join(val.Paths.Parts...)
	}

	return filepath.Join(stepRef.Path...)
}
