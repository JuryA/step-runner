package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"golang.org/x/mod/module"
)

type GitFetcher struct {
	cacheDir string
	mu       sync.Mutex
}

func New(cacheDir string) *GitFetcher {
	return &GitFetcher{cacheDir: cacheDir}
}

func (gf *GitFetcher) Get(ctx context.Context, url, version string) (string, error) {
	gf.mu.Lock()
	defer gf.mu.Unlock()

	endpoint, err := transport.NewEndpoint(url)
	if err != nil {
		return "", fmt.Errorf("parsing git endpoint: %w", err)
	}

	repoDir, err := module.EscapePath(filepath.Join(endpoint.Host, endpoint.Path))
	if err != nil {
		return "", fmt.Errorf("escaping path: %w", err)
	}

	return gf.clone(ctx, repoDir, url, version)
}

func (gf *GitFetcher) clone(ctx context.Context, repoDir, url, version string) (string, error) {
	if version == ".git" {
		// disallow this as a version
		return "", fmt.Errorf(".git version not allowed")
	}

	if err := os.MkdirAll(filepath.Join(gf.cacheDir, repoDir), 0o777); err != nil {
		return "", fmt.Errorf("creating repository directory: %w", err)
	}

	gitDir := filepath.Join(gf.cacheDir, repoDir, ".git")

	var repo *git.Repository

	if _, err := os.Stat(gitDir); err == nil {
		repo, err = git.PlainOpen(gitDir)
		if err != nil {
			return "", fmt.Errorf("opening git repository: %w", err)
		}
	} else {
		repo, err = git.PlainCloneContext(ctx, gitDir, false, &git.CloneOptions{
			Depth:             1,
			URL:               url,
			NoCheckout:        true,
			RecurseSubmodules: git.SubmoduleRescursivity(1),
			ShallowSubmodules: true,
			Mirror:            true,
		})
		if err != nil {
			return "", fmt.Errorf("checking out repository: %w", err)
		}
	}

	// find, fetch, find
	ref := gf.find(repo, version)
	if ref == nil {
		if repo.FetchContext(ctx, &git.FetchOptions{Depth: 1}) == nil {
			ref = gf.find(repo, version)
		}
	}

	if ref == nil {
		return "", fmt.Errorf("cannot find version %q", version)
	}

	ver, err := module.EscapeVersion(ref.Hash().String())
	if err != nil {
		return "", fmt.Errorf("escaping version: %w", err)
	}

	// checkout
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("fetching commit object: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", fmt.Errorf("fetching tree: %w", err)
	}

	worktreeDir, err := os.MkdirTemp(gf.cacheDir, "")
	if err != nil {
		return "", fmt.Errorf("creating temporary worktree dir: %w", err)
	}
	defer os.RemoveAll(worktreeDir)

	if err := tree.Files().ForEach(func(f *object.File) error {
		dest := filepath.Join(worktreeDir, f.Name)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}

		src, err := f.Blob.Reader()
		if err != nil {
			return err
		}
		defer src.Close()

		mode, err := f.Mode.ToOSFileMode()
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		return err
	}); err != nil {
		return "", fmt.Errorf("copying tree: %w", err)
	}

	stepDir := filepath.Join(gf.cacheDir, repoDir, ver)

	// atomic rename
	if err := os.Rename(worktreeDir, stepDir); err != nil {
		if errors.Is(err, os.ErrExist) {
			return stepDir, nil
		}

		return "", fmt.Errorf("renaming dir: %w", err)
	}

	return stepDir, nil
}

func (gf *GitFetcher) find(repo *git.Repository, version string) *plumbing.Reference {
	tag, err := repo.Tag(version)
	if err == nil {
		return tag
	}

	branch, err := repo.Reference(plumbing.NewBranchReferenceName(version), true)
	if err == nil {
		return branch
	}

	partial, err := repo.Reference(plumbing.NewBranchReferenceName("refs/heads/"+version), true)
	if err == nil {
		return partial
	}

	commits, err := repo.CommitObjects()
	if err != nil {
		return nil
	}
	defer commits.Close()

	var found *plumbing.Reference
	_ = commits.ForEach(func(c *object.Commit) error {
		h := c.Hash.String()

		if h == version || (len(version) > 7 && strings.HasPrefix(h, version)) {
			found = plumbing.NewHashReference(plumbing.ReferenceName(h), c.ID())
			return storer.ErrStop
		}

		return nil
	})

	return found
}
