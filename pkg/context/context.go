package context

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

const (
	exportFilename = "export"
)

type Global struct {
	WorkDir    string            `json:"work_dir"`
	Job        map[string]string `json:"job"`
	ExportFile string            `json:"export_file"`
	Env        map[string]string `json:"-"`
	Stdout     io.Writer         `json:"-"`
	Stderr     io.Writer         `json:"-"`

	dir string
}

func NewGlobal() (*Global, error) {
	dir, err := os.MkdirTemp("", "step-runner-export-*")
	if err != nil {
		return nil, fmt.Errorf("making export directory: %w", err)
	}
	exportFile := filepath.Join(dir, exportFilename)
	_, err = os.Create(exportFile)
	if err != nil {
		return nil, fmt.Errorf("creating export file: %w", err)
	}

	return &Global{
		Job:        map[string]string{},
		ExportFile: exportFile,
		Env:        map[string]string{},
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		dir:        dir,
	}, nil
}

func (g *Global) InheritEnv(envs ...string) {
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

func (g *Global) ExportTo(result *proto.StepResult) error {
	exports, err := godotenv.Read(g.ExportFile)
	if err != nil {
		return fmt.Errorf("reading exports: %w", err)
	}
	if result.Exports == nil {
		result.Exports = map[string]string{}
	}
	for k, v := range exports {
		g.Env[k] = v
		result.Exports[k] = v
	}
	err = os.Remove(g.ExportFile)
	if err != nil {
		return fmt.Errorf("clearing export file: %w", err)
	}
	_, err = os.Create(g.ExportFile)
	return err
}

func (g *Global) Cleanup() {
	os.RemoveAll(g.dir)
}

type Steps struct {
	*Global

	StepDir    string                       `json:"step_dir"`
	OutputFile string                       `json:"output_file"`
	Env        map[string]string            `json:"env"`
	Inputs     map[string]*structpb.Value   `json:"inputs"`
	Steps      map[string]*proto.StepResult `json:"steps"`
}

func NewSteps(global *Global) *Steps {
	return &Steps{
		Global: global,
		Env:    maps.Clone(global.Env),
		Inputs: map[string]*structpb.Value{},
		Steps:  map[string]*proto.StepResult{},
	}
}

func (s *Steps) GetEnvs() map[string]string {
	r := make(map[string]string)
	for k, v := range s.Global.Env {
		r[k] = v
	}
	for k, v := range s.Env {
		r[k] = v
	}
	return r
}

func (s *Steps) GetEnvList() []string {
	r := []string{}
	for k, v := range s.GetEnvs() {
		r = append(r, k+"="+v)
	}
	return r
}
