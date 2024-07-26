package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type FileResource struct {
	dir      string
	path     []string
	filename string
}

func NewFileResource(dir string, path []string, filename string) *FileResource {
	return &FileResource{
		dir:      dir,
		path:     path,
		filename: filename,
	}
}

func (l *FileResource) Load(_ context.Context) (string, error) {
	name := filepath.Join(l.path...)
	name = filepath.Join(l.dir, name, l.filename)

	contents, err := os.ReadFile(name)

	if err != nil {
		return "", fmt.Errorf("failed to load resource from file %s: %w", name, err)
	}

	return string(contents), nil
}
