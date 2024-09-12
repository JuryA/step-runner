package bldr

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepsContextBuilder struct {
	globalCtx *runner.GlobalContext
	env       map[string]string
	inputs    map[string]*structpb.Value
}

func StepsContext() *StepsContextBuilder {
	return &StepsContextBuilder{
		globalCtx: GlobalContext().Build(),
		env:       map[string]string{},
		inputs:    map[string]*structpb.Value{},
	}
}

func (bldr *StepsContextBuilder) WithGlobalContext(globalCtx *runner.GlobalContext) *StepsContextBuilder {
	bldr.globalCtx = globalCtx
	return bldr
}

func (bldr *StepsContextBuilder) WithEnv(key, value string) *StepsContextBuilder {
	bldr.env[key] = value
	return bldr
}

func (bldr *StepsContextBuilder) WithInput(name string, value *structpb.Value) *StepsContextBuilder {
	bldr.inputs[name] = value
	return bldr
}

func (bldr *StepsContextBuilder) Build() *runner.StepsContext {
	return &runner.StepsContext{
		GlobalContext: bldr.globalCtx,
		StepDir:       ".",
		OutputFile:    "output",
		Env:           bldr.env,
		Inputs:        bldr.inputs,
		Steps:         map[string]*proto.StepResult{},
	}
}
