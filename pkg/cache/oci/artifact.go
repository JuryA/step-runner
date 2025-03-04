package oci

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type Artifact struct {
	From     string
	To       string
	Platform *v1.Platform
}

func NewArtifact(platform *v1.Platform, from, to string) *Artifact {
	return &Artifact{
		From:     from,
		To:       to,
		Platform: platform,
	}
}

func (a *Artifact) FS() (fs.FS, func() error, error) {
	baseDir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		return nil, func() error { return nil }, fmt.Errorf("create temporary directory: %w", err)
	}

	fromPath := filepath.Clean(a.From)
	toPath := filepath.Join(baseDir, filepath.Clean(a.To))
	toDir, _ := filepath.Split(toPath)
	cleanup := a.removeDir(baseDir)

	fromStat, err := os.Stat(fromPath)
	if err != nil {
		_ = cleanup()
		return nil, func() error { return nil }, err
	}

	if err := os.MkdirAll(toDir, 0755); err != nil {
		_ = cleanup()
		return nil, func() error { return nil }, fmt.Errorf(`create "to" directories: %w`, err)
	}

	switch fromStat.IsDir() {
	case true:
		err = a.copyDir(fromPath, toPath)
	case false:
		err = a.copyFile(fromPath, toPath)
	}

	if err != nil {
		_ = cleanup()
		return nil, func() error { return nil }, err
	}

	return os.DirFS(baseDir), cleanup, nil
}

func (a *Artifact) copyFile(fromPath string, toPath string) error {
	fromFile, err := os.OpenFile(fromPath, os.O_RDONLY, 0000)
	if err != nil {
		return fmt.Errorf(`open "from" file: %w`, err)
	}
	defer fromFile.Close()

	toFile, err := os.OpenFile(toPath, os.O_WRONLY|os.O_CREATE, 0444)
	if err != nil {
		return fmt.Errorf(`open "to" file: %w`, err)
	}
	defer toFile.Close()

	if _, err := io.Copy(toFile, fromFile); err != nil {
		return fmt.Errorf(`copy "from" file to "to" file: %w`, err)
	}

	if err := fromFile.Close(); err != nil {
		return fmt.Errorf(`close "from" file: %w`, err)
	}

	if err := toFile.Close(); err != nil {
		return fmt.Errorf(`close "to" file: %w`, err)
	}

	return nil
}

func (a *Artifact) removeDir(toDir string) func() error {
	return func() error {
		if err := os.RemoveAll(toDir); err != nil {
			return fmt.Errorf(`remove "to" directory %q: %w`, toDir, err)
		}

		return nil
	}
}

func (a *Artifact) copyDir(fromPath string, toPath string) error {
	if err := os.CopyFS(toPath, os.DirFS(fromPath)); err != nil {
		return fmt.Errorf(`copy "from" dir to "to" dir: %w`, err)
	}

	return nil
}

func (a *Artifact) String() string {
	return fmt.Sprintf("%s[%s->%s]", a.Platform, a.From, a.To)
}
