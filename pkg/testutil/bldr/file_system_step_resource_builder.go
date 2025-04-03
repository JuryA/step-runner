package bldr

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

type FileSystemStepResourceBuilder struct {
	dir      string
	filename string
	t        *testing.T
}

func FileSystemStepResource(t *testing.T) *FileSystemStepResourceBuilder {
	return &FileSystemStepResourceBuilder{
		t:        t,
		dir:      t.TempDir(),
		filename: "step.yml",
	}
}

func (bldr *FileSystemStepResourceBuilder) WithDir(dir string) *FileSystemStepResourceBuilder {
	bldr.dir = dir
	return bldr
}

func (bldr *FileSystemStepResourceBuilder) Build() *runner.FileSystemStepResource {
	stepFile := filepath.Join(bldr.dir, bldr.filename)

	if _, err := os.Stat(stepFile); errors.Is(err, os.ErrNotExist) {
		err = os.WriteFile(stepFile, []byte("spec:\n---\nexec: {command: [sh]}"), 0644)
		require.NoError(bldr.t, err)
	}

	return runner.NewFileSystemStepResource(bldr.dir, "step.yml")
}
