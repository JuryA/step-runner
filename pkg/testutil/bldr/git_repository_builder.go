package bldr

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/require"
)

type GitRepositoryBuilder struct {
	initWithFilesFromPath string
}

func GitRepository() *GitRepositoryBuilder {
	return &GitRepositoryBuilder{
		initWithFilesFromPath: "",
	}
}

func (bldr *GitRepositoryBuilder) InitWithFilesFromDir(path string) *GitRepositoryBuilder {
	bldr.initWithFilesFromPath = path
	return bldr
}

func (bldr *GitRepositoryBuilder) Build(t *testing.T) (*git.Repository, *git.Worktree) {
	repo, err := git.InitWithOptions(memory.NewStorage(), memfs.New(), git.InitOptions{DefaultBranch: plumbing.Main})
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	GitWorktree(t, worktree).
		CreateFile("README.md", "Git repository").
		Stage("README.md").
		Commit("Initial commit")

	if bldr.initWithFilesFromPath != "" {
		GitWorktree(t, worktree).
			CopyFromDir(bldr.initWithFilesFromPath).
			Stage(".").
			Commit("Seed initial files")
	}

	return repo, worktree
}
