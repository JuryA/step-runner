package runner

import (
	"fmt"
	"io"
	"os"
)

type GlobalContext struct {
	WorkDir    string
	Job        map[string]string
	ExportFile *StepFile
	Env        *Environment
	Stdout     io.Writer
	Stderr     io.Writer
}

func NewGlobalContext(env *Environment) (*GlobalContext, error) {
	exportFile, err := NewStepFileInTmp()

	if err != nil {
		return nil, fmt.Errorf("failed to create export file: %w", err)
	}

	return &GlobalContext{
		Job:        map[string]string{},
		ExportFile: exportFile,
		Env:        env,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}, nil
}

func (g *GlobalContext) Exports() (map[string]string, error) {
	exports, err := g.ExportFile.ReadDotEnv()
	if err != nil {
		return nil, fmt.Errorf("reading exports: %w", err)
	}

	g.Env = g.Env.AddLexicalScope(exports)

	err = g.ExportFile.Remove()
	if err != nil {
		return nil, fmt.Errorf("clearing export file: %w", err)
	}

	exportFile, err := NewStepFileInTmp()

	if err != nil {
		return nil, fmt.Errorf("failed to create export file: %w", err)
	}

	g.ExportFile = exportFile
	return exports, nil
}

func (g *GlobalContext) Cleanup() {
	_ = g.ExportFile.Remove()
}
