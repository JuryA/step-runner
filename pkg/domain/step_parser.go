package domain

import "gitlab.com/gitlab-org/step-runner/proto"

type StepParser interface {
	Parse(rawSteps string) (Step, *proto.SpecDefinition, error)
}
