package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type LocalFileLoader struct {
	dir      string
	path     []string
	filename string
}

func NewLocalFileLoader(dir string, path []string, filename string) *LocalFileLoader {
	return &LocalFileLoader{
		dir:      dir,
		path:     path,
		filename: filename,
	}
}

func (l *LocalFileLoader) Load(_ context.Context) ([]byte, error) {
	name := filepath.Join(l.path...)
	name = filepath.Join(l.dir, name, l.filename)

	contents, err := os.ReadFile(name)

	if err != nil {
		return nil, fmt.Errorf("failed to load resource from file %s: %w", name, err)
	}

	return contents, nil
}
