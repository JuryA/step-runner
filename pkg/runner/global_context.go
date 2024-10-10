package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const (
	exportFilename = "export"
)

type GlobalContext struct {
	WorkDir    string
	Job        map[string]string
	ExportFile string
	Env        *Environment
	Stdout     io.Writer
	Stderr     io.Writer

	dir string
}

func NewGlobalContext(env *Environment) (*GlobalContext, error) {
	dir, err := os.MkdirTemp("", "step-runner-export-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create global context: failed to make export directory: %w", err)
	}
	exportFile := filepath.Join(dir, exportFilename)
	_, err = os.Create(exportFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create global context: failed to create export file: %w", err)
	}

	return &GlobalContext{
		Job:        map[string]string{},
		ExportFile: exportFile,
		Env:        env,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		dir:        dir,
	}, nil
}

func (g *GlobalContext) Exports() (map[string]string, error) {
	exports, err := godotenv.Read(g.ExportFile)
	if err != nil {
		return nil, fmt.Errorf("reading exports: %w", err)
	}

	g.Env = g.Env.AddLexicalScope(exports)

	err = os.Remove(g.ExportFile)
	if err != nil {
		return nil, fmt.Errorf("clearing export file: %w", err)
	}
	_, err = os.Create(g.ExportFile)
	return exports, err
}

func (g *GlobalContext) Cleanup() {
	os.RemoveAll(g.dir)
}
