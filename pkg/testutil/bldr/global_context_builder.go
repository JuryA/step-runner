package bldr

import (
	"bytes"
	"os"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type GlobalContextBuilder struct {
	job        map[string]string
	exportFile *runner.StepFile
}

func GlobalContext() *GlobalContextBuilder {
	return &GlobalContextBuilder{
		job:        map[string]string{},
		exportFile: nil,
	}
}

func (bldr *GlobalContextBuilder) WithJob(name, value string) *GlobalContextBuilder {
	bldr.job[name] = value
	return bldr
}

func (bldr *GlobalContextBuilder) WithTempExportFile(tempDir string) *GlobalContextBuilder {
	exportFile, err := runner.NewStepFileInDir(tempDir)

	if err != nil {
		panic(err)
	}

	bldr.exportFile = exportFile
	return bldr
}

func (bldr *GlobalContextBuilder) Build() *runner.GlobalContext {
	if bldr.exportFile == nil {
		bldr.WithTempExportFile(os.TempDir())
	}

	return &runner.GlobalContext{
		WorkDir:    ".",
		Job:        bldr.job,
		ExportFile: bldr.exportFile,
		Env:        runner.NewEmptyEnvironment(),
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
	}
}
