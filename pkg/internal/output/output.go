package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

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
	stepCtx    *context.Steps
	dir        string
	outputFile string
	exportFile string
}

func New(stepCtx *context.Steps) (*Files, error) {
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
	stepCtx.Env[outputFileKey] = outputFile
	stepCtx.Env[exportFileKey] = exportFile
	return &Files{
		stepCtx:    stepCtx,
		dir:        dir,
		outputFile: outputFile,
		exportFile: exportFile,
	}, nil
}

func (f *Files) OutputTo(result *proto.StepResult) error {
	outputs, err := godotenv.Read(f.outputFile)
	if err != nil {
		return fmt.Errorf("reading outputs: %w", err)
	}
	result.Outputs = outputs
	return nil
}

func (f *Files) ExportTo(globalCtx *context.Global, result *proto.StepResult) error {
	exports, err := godotenv.Read(f.exportFile)
	if err != nil {
		return fmt.Errorf("reading exports: %w", err)
	}
	for k, v := range exports {
		globalCtx.Env[k] = v
	}
	result.Exports = exports
	return nil
}

func (f *Files) Cleanup() {
	os.RemoveAll(f.dir)
}
