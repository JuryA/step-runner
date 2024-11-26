package runner

import "gitlab.com/gitlab-org/step-runner/proto"

// StepReference knows how the step was loaded
type StepReference interface {
	ToProtoStep(*Params) *proto.Step
	Describer
}

// StepDefinedInGitLabJob is a step defined in a GitLab jobs using the run: syntax or STEPS: variable
var StepDefinedInGitLabJob = NewNamedStepReference("", nil)
