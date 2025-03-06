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
	Src      string
	Dst      string
	Platform *v1.Platform
}

func NewArtifact(platform *v1.Platform, src, dst string) *Artifact {
	return &Artifact{
		Src:      src,
		Dst:      dst,
		Platform: platform,
	}
}

func (a *Artifact) FS() (fs.FS, func() error, error) {
	baseDir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		return nil, func() error { return nil }, fmt.Errorf("create temporary directory: %w", err)
	}

	src := filepath.Clean(a.Src)
	dst := filepath.Join(baseDir, filepath.Clean(a.Dst))
	dstDir, _ := filepath.Split(dst)
	cleanup := a.removeDir(baseDir)

	statSrc, err := os.Stat(src)
	if err != nil {
		_ = cleanup()
		return nil, func() error { return nil }, err
	}

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		_ = cleanup()
		return nil, func() error { return nil }, fmt.Errorf(`create destination directories: %w`, err)
	}

	switch statSrc.IsDir() {
	case true:
		err = a.copyDir(src, dst)
	case false:
		err = a.copyFile(src, dst)
	}

	if err != nil {
		_ = cleanup()
		return nil, func() error { return nil }, err
	}

	return os.DirFS(baseDir), cleanup, nil
}

func (a *Artifact) copyFile(src string, dst string) error {
	srcFile, err := os.OpenFile(src, os.O_RDONLY, 0000)
	if err != nil {
		return fmt.Errorf(`open source file: %w`, err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, 0444)
	if err != nil {
		return fmt.Errorf(`open destination file: %w`, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf(`copy source file to destination file: %w`, err)
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf(`close destination file: %w`, err)
	}

	return nil
}

func (a *Artifact) removeDir(dstDir string) func() error {
	return func() error {
		if err := os.RemoveAll(dstDir); err != nil {
			return fmt.Errorf(`remove destination directory %q: %w`, dstDir, err)
		}

		return nil
	}
}

func (a *Artifact) copyDir(src string, dst string) error {
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		return fmt.Errorf(`copy source dir to destination dir: %w`, err)
	}

	return nil
}

func (a *Artifact) String() string {
	return fmt.Sprintf("%s[%s->%s]", a.Platform, a.Src, a.Dst)
}
