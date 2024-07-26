package di

import (
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/git"
)

func InitializeGitFetcher() func(*Container) error {
	return func(c *Container) error {
		cacheDir := filepath.Join(os.TempDir(), "step-runner-cache")

		if err := os.MkdirAll(cacheDir, 0o750); err != nil {
			return fmt.Errorf("failed to create cache dir %q: %w", cacheDir, err)
		}

		c.GitFetcher = git.New(cacheDir)
		return nil
	}
}
