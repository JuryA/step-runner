package runner

import "gitlab.com/gitlab-org/step-runner/proto"

type StepParser interface {
	Parse(*proto.SpecDefinition) (Step, error)
}
