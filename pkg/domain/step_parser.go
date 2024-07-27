package domain

import "gitlab.com/gitlab-org/step-runner/proto"

type StepParser interface {
	Parse(yamlSteps string, dir string) (Step, *proto.SpecDefinition, error)
}
