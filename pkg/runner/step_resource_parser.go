package runner

import (
	"fmt"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepResourceParser struct {
	gitFetcher  *git.GitFetcher
	ociFetcher  *oci.OCIFetcher
	distFetcher *dist.Fetcher
}

func NewStepResourceParser(gitFetcher *git.GitFetcher, ociFetcher *oci.OCIFetcher, distFetcher *dist.Fetcher) *StepResourceParser {
	return &StepResourceParser{
		gitFetcher:  gitFetcher,
		ociFetcher:  ociFetcher,
		distFetcher: distFetcher,
	}
}

func (p *StepResourceParser) Parse(parentDir string, stepRef *proto.Step_Reference) (StepResource, error) {
	stepDir := filepath.Join(stepRef.Path...)

	switch stepRef.Protocol {
	case proto.StepReferenceProtocol_local:
		dir := stepDir
		if !strings.HasPrefix(stepDir, "/") {
			dir = filepath.Join(parentDir, stepDir)
		}

		return NewFileSystemStepResource(dir, stepRef.Filename), nil

	case proto.StepReferenceProtocol_git:
		return NewGitStepResource(p.gitFetcher, stepRef.Url, stepRef.Version, stepDir, stepRef.Filename), nil

	case proto.StepReferenceProtocol_oci:
		return NewOCIStepResource(p.ociFetcher, stepRef.Registry, stepRef.Repository, stepRef.Tag, stepDir, stepRef.Filename), nil

	case proto.StepReferenceProtocol_dist:
		return NewDistStepResource(p.distFetcher, stepDir, stepRef.Filename), nil

	case proto.StepReferenceProtocol_dynamic:
		return NewDynamicStepResource(p, stepRef.Url), nil

	case proto.StepReferenceProtocol_spec_def:
		return NewFixedStepResource(stepRef.SpecDef), nil
	}

	return nil, fmt.Errorf("unknown step reference protocol: %s", stepRef.Protocol)
}
