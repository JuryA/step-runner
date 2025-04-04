package dist

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gitlab.com/gitlab-org/step-runner/dist"
)

type Fetcher struct {
	workDirMu   sync.Mutex
	workDir     string
	stepsFinder dist.StepFinder
}

func NewFetcher(stepsFinder dist.StepFinder) *Fetcher {
	return &Fetcher{
		stepsFinder: stepsFinder,
	}
}

func (f *Fetcher) Fetch(step string) (string, error) {
	f.workDirMu.Lock()
	defer f.workDirMu.Unlock()

	workDir, err := f.createWorkDir()
	if err != nil {
		return "", fmt.Errorf("fetch dist step %s: %w", step, err)
	}

	stepDirFS, err := f.stepsFinder(step)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}

	downloadDir := filepath.Join(workDir, step)

	if _, err := os.Stat(downloadDir); err == nil {
		return workDir, nil
	}

	if err := os.CopyFS(downloadDir, stepDirFS); err != nil {
		return "", fmt.Errorf("fetch dist step %s: copy: %w", step, err)
	}

	if err := f.chmodFiles(downloadDir); err != nil {
		return "", fmt.Errorf("fetch dist step %s: %w", step, err)
	}

	return workDir, nil
}

func (f *Fetcher) CleanUp() {
	f.workDirMu.Lock()
	defer f.workDirMu.Unlock()

	_ = os.RemoveAll(f.workDir)
	f.workDir = ""
}

func (f *Fetcher) createWorkDir() (string, error) {
	if f.workDir == "" {
		tempDir, err := os.MkdirTemp("", "")
		if err != nil {
			return "", fmt.Errorf("creating work dir: %w", err)
		}

		f.workDir = tempDir
	}

	return f.workDir, nil
}

func (f *Fetcher) chmodFiles(stepDir string) error {
	permissions := map[string]fs.FileMode{}

	err := filepath.WalkDir(stepDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		switch {
		case d.IsDir():
			permissions[path] = 0755
		case d.Name() == "run" ||
			strings.HasSuffix(path, ".exe") ||
			strings.HasSuffix(path, ".sh"):
			permissions[path] = 0555
		default:
			permissions[path] = 0444
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("determine file permissions: %w", err)
	}

	for path, mode := range permissions {
		if err := os.Chmod(path, os.FileMode(mode)); err != nil {
			return err
		}
	}

	return nil
}
