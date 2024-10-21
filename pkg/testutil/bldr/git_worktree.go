package bldr

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

type GitWorktreeBuilder struct {
	worktree *git.Worktree
	t        *testing.T
}

func GitWorktree(t *testing.T, worktree *git.Worktree) *GitWorktreeBuilder {
	return &GitWorktreeBuilder{
		t:        t,
		worktree: worktree,
	}
}

func (bldr *GitWorktreeBuilder) CreateFile(path string, content string) *GitWorktreeBuilder {
	file, err := bldr.worktree.Filesystem.Create(path)
	require.NoError(bldr.t, err)
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, strings.NewReader(content))
	require.NoError(bldr.t, err)
	return bldr
}

func (bldr *GitWorktreeBuilder) MakeDir(dir string) *GitWorktreeBuilder {
	err := bldr.worktree.Filesystem.MkdirAll(dir, 0o755)
	require.NoError(bldr.t, err)
	return bldr
}

func (bldr *GitWorktreeBuilder) CopyFromDir(fromDir string) *GitWorktreeBuilder {
	err := filepath.Walk(fromDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		withoutOldDir, _ := strings.CutPrefix(path, fromDir)
		newPath, _ := strings.CutPrefix(withoutOldDir, "/")

		if newPath == "" {
			return nil
		}

		if info.IsDir() {
			return bldr.worktree.Filesystem.MkdirAll(newPath, info.Mode())
		}

		fromFile, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open from file: %w", err)
		}
		defer func() { _ = fromFile.Close() }()

		toFile, err := bldr.worktree.Filesystem.OpenFile(newPath, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, info.Mode())
		if err != nil {
			return fmt.Errorf("failed to open new file: %w", err)
		}
		defer func() { _ = toFile.Close() }()

		_, err = io.Copy(toFile, fromFile)
		if err != nil {
			return fmt.Errorf("failed to copy from old to new file: %w", err)
		}

		return nil
	})

	require.NoError(bldr.t, err)
	return bldr
}

func (bldr *GitWorktreeBuilder) Stage(glob string) *GitWorktreeBuilder {
	err := bldr.worktree.AddGlob(glob)
	require.NoError(bldr.t, err)
	return bldr
}

func (bldr *GitWorktreeBuilder) Commit(commitMsg string) string {
	opts := &git.CommitOptions{
		AllowEmptyCommits: false,
		Author: &object.Signature{
			Name:  "Step Runner Test",
			Email: "step_runner_test@gitlab.com",
			When:  time.Now(),
		},
	}
	commit, err := bldr.worktree.Commit(commitMsg, opts)
	require.NoError(bldr.t, err)
	return commit.String()
}
