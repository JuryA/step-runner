package steps

import (
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/steps/action"
)

var InlineSteps = map[string]runner.InlineStepFn{
	"inline-step://action": action.Run,
}
