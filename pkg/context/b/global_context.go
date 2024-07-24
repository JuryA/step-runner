package b

import (
	"bytes"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
)

type GlobalContextBuilder struct {
}

func GlobalContext() *GlobalContextBuilder {
	return &GlobalContextBuilder{}
}

func (*GlobalContextBuilder) Build() *context.Global {
	return &context.Global{
		WorkDir:    ".",
		Job:        map[string]string{},
		ExportFile: "export",
		Env:        map[string]string{},
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
	}
}
