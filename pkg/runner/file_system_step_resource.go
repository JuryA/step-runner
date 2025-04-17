package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

// FileSystemStepResource knows how to load a step from the file system using an absolute path
type FileSystemStepResource struct {
	workDir  string
	stepPath string
	filename string
}

func NewFileSystemStepResource(workDir string, stepPath, filename string) *FileSystemStepResource {
	return &FileSystemStepResource{
		workDir:  workDir,
		stepPath: stepPath,
		filename: filename,
	}
}

func (sr *FileSystemStepResource) Fetch(_ context.Context, view *expression.InterpolationContext) (*SpecDefinition, error) {
	stepDir, err := sr.stepDir(view)
	if err != nil {
		return nil, fmt.Errorf("fetching step from file: %w", err)
	}

	stepFile := filepath.Join(stepDir, sr.filename)

	spec, step, err := schema.LoadSteps(stepFile)
	if err != nil {
		return nil, fmt.Errorf("loading file %q: %w", stepFile, err)
	}

	protoSpec, err := spec.Compile()
	if err != nil {
		return nil, fmt.Errorf("compiling proto specification: %w", err)
	}

	protoDef, err := step.Compile()
	if err != nil {
		return nil, fmt.Errorf("compiling proto definition: %w", err)
	}

	return NewSpecDefinition(protoSpec, protoDef, stepDir), nil
}

func (sr *FileSystemStepResource) stepDir(view *expression.InterpolationContext) (string, error) {
	stepPath, err := expression.ExpandString(view, sr.stepPath)
	if err != nil {
		return "", fmt.Errorf("expanding step path %q: %w", stepPath, err)
	}

	if strings.HasPrefix(stepPath, "/") {
		return stepPath, nil
	}

	return filepath.Join(sr.workDir, stepPath), nil
}

func (sr *FileSystemStepResource) Describe() string {
	return filepath.Join(sr.workDir, sr.stepPath, sr.filename)
}
