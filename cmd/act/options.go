package act

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type Options struct {
	CLIStepsContext *CLIStepsContext
}

func NewOptions() *Options {
	return &Options{
		CLIStepsContext: &CLIStepsContext{},
	}
}

func (o *Options) Validate() error {
	if o.CLIStepsContext == nil {
		return fmt.Errorf("steps-context is required")
	}

	return nil
}

func (o *Options) ToStepsContext() (*runner.StepsContext, error) {
	job := make(map[string]string)

	for _, v := range o.CLIStepsContext.StepsContext.Job {
		job[v.Key] = v.Value
	}

	globalCtx := runner.NewGlobalContext(runner.NewEmptyEnvironment())
	globalCtx.Job = job
	globalCtx.WorkDir = o.CLIStepsContext.StepsContext.WorkDir

	stepDir := o.CLIStepsContext.StepsContext.StepDir
	inputs := o.CLIStepsContext.StepsContext.Inputs
	env := runner.NewEnvironment(o.CLIStepsContext.StepsContext.Env)

	return runner.NewStepsContext(globalCtx, stepDir, inputs, env)
}
