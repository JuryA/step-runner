package runner

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

const (
	exportFilename = "export"
)

type GlobalContext struct {
	WorkDir    string            `json:"work_dir"`
	Job        map[string]string `json:"job"`
	ExportFile string            `json:"export_file"`
	Env        map[string]string `json:"-"`
	Stdout     io.Writer         `json:"-"`
	Stderr     io.Writer         `json:"-"`

	dir string
}

func NewGlobalContext() (*GlobalContext, error) {
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
		Env:        map[string]string{},
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		dir:        dir,
	}, nil
}

func (g *GlobalContext) InheritEnv(envs ...string) {
	if g.Env == nil {
		g.Env = make(map[string]string, len(envs))
	}
	for _, e := range envs {
		k, v, ok := strings.Cut(e, "=")
		if ok {
			g.Env[k] = v
		}
	}
}

func (g *GlobalContext) Exports() (map[string]string, error) {
	exports, err := godotenv.Read(g.ExportFile)
	if err != nil {
		return nil, fmt.Errorf("reading exports: %w", err)
	}

	for k, v := range exports {
		g.Env[k] = v
	}
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

func (g *GlobalContext) NewEnvMergedFrom(env map[string]string) map[string]string {
	merged := maps.Clone(g.Env)
	maps.Copy(merged, env)
	return merged
}
