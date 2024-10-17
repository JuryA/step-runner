package bldr

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
)

type InterpolationCtxBuilder struct {
	env map[string]string
}

func InterpolationCtx() *InterpolationCtxBuilder {
	return &InterpolationCtxBuilder{
		env: map[string]string{},
	}
}

func (bldr *InterpolationCtxBuilder) WithEnvVar(name, value string) *InterpolationCtxBuilder {
	bldr.env[name] = value
	return bldr
}

func (bldr *InterpolationCtxBuilder) Build() *expression.InterpolationContext {
	return &expression.InterpolationContext{
		Env:         bldr.env,
		ExportFile:  "export_file",
		Inputs:      map[string]*structpb.Value{},
		Job:         map[string]string{},
		OutputFile:  "output_file",
		StepDir:     "step.dir",
		StepResults: map[string]*expression.StepResultView{},
		WorkDir:     "work.dir",
	}
}
