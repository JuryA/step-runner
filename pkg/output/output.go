package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

const (
	outputFilename = "output"
	outputFileKey  = "STEP_RUNNER_OUTPUT"
	exportFilename = "export"
	exportFileKey  = "STEP_RUNNER_ENV"
)

type Files struct {
	step       *proto.Step
	dir        string
	outputFile string
	exportFile string
}

func New(step *proto.Step) (*Files, error) {
	dir, err := os.MkdirTemp("", "step-runner-output-*")
	if err != nil {
		return nil, fmt.Errorf("making output directoy: %w", err)
	}
	outputFile := filepath.Join(dir, outputFilename)
	err = os.WriteFile(outputFile, []byte{}, 0660)
	if err != nil {
		return nil, fmt.Errorf("creating output file: %w", err)
	}
	exportFile := filepath.Join(dir, exportFilename)
	err = os.WriteFile(exportFile, []byte{}, 0660)
	if err != nil {
		return nil, fmt.Errorf("creating export file: %w", err)
	}
	if step.Env == nil {
		step.Env = map[string]string{}
	}
	step.Env[outputFileKey] = outputFile
	step.Env[exportFileKey] = exportFile
	return &Files{
		step:       step,
		dir:        dir,
		outputFile: outputFile,
		exportFile: exportFile,
	}, nil
}

func (f *Files) OutputTo(stepCtx *context.Steps, result *proto.StepResult) error {
	outputs, err := readFile(f.outputFile)
	if err != nil {
		return fmt.Errorf("reading outputs: %w", err)
	}
	stepCtx.Outputs[f.step.Name] = outputs
	result.Outputs = outputs
	return nil
}

func (f *Files) ExportTo(globalCtx *context.Global, result *proto.StepResult) error {
	exports, err := readFile(f.exportFile)
	if err != nil {
		return fmt.Errorf("reading exports: %w", err)
	}
	for k, v := range exports {
		globalCtx.Env[k] = v
	}
	result.Exports = exports
	return nil
}

func (f *Files) Cleanup(result *proto.StepResult) {
	delete(f.step.Env, outputFileKey)
	delete(f.step.Env, exportFileKey)
	if result != nil {
		delete(result.Step.Env, outputFileKey)
		delete(result.Step.Env, exportFileKey)
	}
	os.RemoveAll(f.dir)
}

func readFile(filename string) (map[string]string, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file %v: %w", filename, err)
	}
	out := map[string]string{}
	lines := strings.Split(string(bytes), "\n")
	for _, l := range lines {
		if len(l) == 0 {
			continue
		}
		fields := strings.Split(l, "=")
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid line %q", l)
		}
		key := fields[0]
		value := l[len(key)+1:]
		out[key] = value
	}
	return out, nil
}
