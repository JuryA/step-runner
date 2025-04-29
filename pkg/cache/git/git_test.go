package git_test

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"

	gitFetch "gitlab.com/gitlab-org/step-runner/pkg/cache/git"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestGitFetcher(t *testing.T) {
	tests := map[string]struct {
		version          string
		path             []string
		expectClonedFile string
		modifyRepo       func(t *testing.T, repo *git.Repository) string
	}{
		"clone using the default branch": {
			version:          "main",
			expectClonedFile: "step.yml",
			path:             []string{"steps", "echo"},
			modifyRepo:       func(t *testing.T, repo *git.Repository) string { return "" },
		},
		"clone using a branch": {
			version:          "my-branch",
			expectClonedFile: "step.yml",
			path:             []string{"steps", "echo"},
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
			path:             []string{"steps", "echo"},
			modifyRepo: func(t *testing.T, repo *git.Repository) string {
				worktree, err := repo.Worktree()
				require.NoError(t, err)

				return bldr.GitWorktree(t, worktree).
					CreateFile(path.Join("steps", "echo", "file.txt"), "data").
					Stage(path.Join("steps", "echo", "file.txt")).
					Commit("Add text file")
			},
		},
		"clone using first eight-letters of commit hash": {
			version:          "<use commit hash returned from modifyRepo>",
			expectClonedFile: "file.txt",
			path:             []string{"steps", "echo"},
			modifyRepo: func(t *testing.T, repo *git.Repository) string {
				worktree, err := repo.Worktree()
				require.NoError(t, err)

				commit := bldr.GitWorktree(t, worktree).
					CreateFile(path.Join("steps", "echo", "file.txt"), "data").
					Stage(path.Join("steps", "echo", "file.txt")).
					Commit("Add text file")
				return commit[:8]
			},
		},
		"clone using a lightweight tag": {
			version:          "v1.0.0",
			expectClonedFile: "step.yml",
			path:             []string{"steps", "echo"},
			modifyRepo: func(t *testing.T, repo *git.Repository) string {
				head, err := repo.Head()
				require.NoError(t, err)

				_, err = repo.CreateTag("v1.0.0", head.Hash(), nil)
				require.NoError(t, err)
				return ""
			},
		},
		"clone using an annotated tag": {
			version:          "v1.0.0",
			expectClonedFile: "step.yml",
			path:             []string{"steps", "echo"},
			modifyRepo: func(t *testing.T, repo *git.Repository) string {
				head, err := repo.Head()
				require.NoError(t, err)

				opts := &git.CreateTagOptions{
					Message: "tag msg",
					Tagger: &object.Signature{
						Name:  "Sally Seashells",
						Email: "sally-seashells@gitlab+fake.com",
						When:  time.Now(),
					},
				}
				_, err = repo.CreateTag("v1.0.0", head.Hash(), opts)
				require.NoError(t, err)
				return ""
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, _ := bldr.GitRepository().InitWithFilesFromDir("../../../e2e_tests").Build(t)
			gitServerURL := bldr.StartGitSmartHTTPServer(t, repo)

			hash := test.modifyRepo(t, repo)
			version := test.version

			if hash != "" {
				version = hash
			}

			fetcher := gitFetch.New(t.TempDir(), gitFetch.CloneOptions{Depth: 0})
			clonedDir, err := fetcher.Get(context.Background(), gitServerURL, version)
			require.NoError(t, err)
			require.FileExists(t, path.Join(clonedDir, path.Join(test.path...), test.expectClonedFile))
		})
	}
}

func TestGitFetcher_Caching(t *testing.T) {
	t.Run("clone repository that has been cloned before", func(t *testing.T) {
		repo, _ := bldr.GitRepository().InitWithFilesFromDir("../../../e2e_tests/steps/echo").Build(t)

		head, err := repo.Head()
		require.NoError(t, err)

		_, err = repo.CreateTag("main-copy", head.Hash(), nil)
		require.NoError(t, err)

		gitServerURL := bldr.StartGitSmartHTTPServer(t, repo)
		fetcher := gitFetch.New(t.TempDir(), gitFetch.CloneOptions{Depth: 0})

		for _, version := range []string{"main", "main-copy"} {
			clonedDir, err := fetcher.Get(context.Background(), gitServerURL, version)
			require.NoError(t, err)
			require.FileExists(t, path.Join(clonedDir, "step.yml"))
		}
	})

	t.Run("fetch when previously cloned repository is missing version", func(t *testing.T) {
		repo, worktree := bldr.GitRepository().InitWithFilesFromDir("../../../e2e_tests/steps/echo").Build(t)
		gitServerURL := bldr.StartGitSmartHTTPServer(t, repo)
		fetcher := gitFetch.New(t.TempDir(), gitFetch.CloneOptions{Depth: 0})

		_, err := fetcher.Get(context.Background(), gitServerURL, "main")
		require.NoError(t, err)

		bldr.GitWorktree(t, worktree).
			CreateFile("file.txt", "data").
			Stage("file.txt").
			Commit("Add text file")

		head, err := repo.Head()
		require.NoError(t, err)

		_, err = repo.CreateTag("v1.0.0", head.Hash(), nil)
		require.NoError(t, err)

		clonedDir, err := fetcher.Get(context.Background(), gitServerURL, "v1.0.0")
		require.NoError(t, err)
		require.FileExists(t, path.Join(clonedDir, "file.txt"))
	})
}
