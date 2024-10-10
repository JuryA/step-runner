package bldr

import (
	"bytes"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type GlobalContextBuilder struct {
	job        map[string]string
	exportFile string
}

func GlobalContext() *GlobalContextBuilder {
	return &GlobalContextBuilder{
		job:        map[string]string{},
		exportFile: "export",
	}
}

func (bldr *GlobalContextBuilder) WithJob(name, value string) *GlobalContextBuilder {
	bldr.job[name] = value
	return bldr
}

func (bldr *GlobalContextBuilder) WithTempExportFile(tempDir string) *GlobalContextBuilder {
	exportFile := filepath.Join(tempDir, "export")
	_, err := os.Create(exportFile)

	if err != nil {
		panic(err)
	}

	bldr.exportFile = exportFile
	return bldr
}

func (bldr *GlobalContextBuilder) Build() *runner.GlobalContext {
	return &runner.GlobalContext{
		WorkDir:    ".",
		Job:        bldr.job,
		ExportFile: bldr.exportFile,
		Env:        runner.NewEmptyEnvironment(),
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
	}
}
