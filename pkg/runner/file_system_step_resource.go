package runner

import (
	"context"
	"fmt"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

// FileSystemStepResource knows how to load a step from the file system using an absolute path
type FileSystemStepResource struct {
	dir      string
	filename string
}

func NewFileSystemStepResource(dir string, filename string) *FileSystemStepResource {
	return &FileSystemStepResource{
		dir:      dir,
		filename: filename,
	}
}

func (sr *FileSystemStepResource) Fetch(_ context.Context, _ *expression.InterpolationContext) (*proto.SpecDefinition, error) {
	stepFile := filepath.Join(sr.dir, sr.filename)

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

	protoStepDef := &proto.SpecDefinition{
		Spec:       protoSpec,
		Definition: protoDef,
		Dir:        sr.dir,
	}
	return protoStepDef, nil
}

func (sr *FileSystemStepResource) Describe() string {
	return filepath.Join(sr.dir, sr.filename)
}
