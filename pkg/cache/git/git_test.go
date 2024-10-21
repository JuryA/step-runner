package git_test

import (
	"context"
	"path"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"

	gitFetch "gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestGitFetcher(t *testing.T) {
	tests := map[string]struct {
		version          string
		expectClonedFile string
		modifyRepo       func(t *testing.T, repo *git.Repository) string
	}{
		"clone using the default branch": {
			version:          "main",
			expectClonedFile: "step.yml",
			modifyRepo:       func(t *testing.T, repo *git.Repository) string { return "" },
		},
		"clone using a branch": {
			version:          "my-branch",
			expectClonedFile: "step.yml",
			modifyRepo: func(t *testing.T, repo *git.Repository) string {
				head, err := repo.Head()
				require.NoError(t, err)

				branch := plumbing.NewHashReference("refs/heads/my-branch", head.Hash())
				err = repo.Storer.SetReference(branch)
				require.NoError(t, err)
				return ""
			},
		},
		"clone using a commit hash": {
			version:          "<use commit hash returned from modifyRepo>",
			expectClonedFile: "file.txt",
			modifyRepo: func(t *testing.T, repo *git.Repository) string {
				worktree, err := repo.Worktree()
				require.NoError(t, err)

				return bldr.GitWorktree(t, worktree).
					CreateFile("file.txt", "data").
					Stage("file.txt").
					Commit("Add text file")
			},
		},
		"clone using a lightweight tag": {
			version:          "v1.0.0",
			expectClonedFile: "step.yml",
			modifyRepo: func(t *testing.T, repo *git.Repository) string {
				head, err := repo.Head()
				require.NoError(t, err)

				_, err = repo.CreateTag("v1.0.0", head.Hash(), nil)
				require.NoError(t, err)
				return ""
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, _ := bldr.GitRepository().InitWithFilesFromDir("../../runner/test_steps/echo").Build(t)
			gitServerURL := bldr.StartGitSmartHTTPServer(t, repo)

			hash := test.modifyRepo(t, repo)
			version := test.version

			if hash != "" {
				version = hash
			}

			fetcher := gitFetch.New(t.TempDir(), gitFetch.CloneOptions{Depth: 0})
			clonedDir, err := fetcher.Get(context.Background(), gitServerURL, version)
			require.NoError(t, err)
			require.FileExists(t, path.Join(clonedDir, test.expectClonedFile))
		})
	}
}
