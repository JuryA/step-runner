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
	options  CloneOptions
	mu       sync.Mutex
}

type CloneOptions struct {
	Depth int
}

func New(cacheDir string, options CloneOptions) *GitFetcher {
	return &GitFetcher{cacheDir: cacheDir, options: options}
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
			Depth:             gf.options.Depth,
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

	ref, err := gf.findFetchAndFindAgain(ctx, repo, version)
	if err != nil {
		return "", err
	}

	if ref == nil {
		return "", fmt.Errorf("cannot find version %q", version)
	}

	ver, err := ref.EscapeHash()
	if err != nil {
		return "", fmt.Errorf("escaping version: %w", err)
	}

	// checkout
	commit, err := ref.Commit(repo)
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

func (gf *GitFetcher) findFetchAndFindAgain(ctx context.Context, repo *git.Repository, version string) (*Reference, error) {
	ref, err := gf.find(repo, version)

	if err != nil {
		return nil, fmt.Errorf("finding version %q: %w", version, err)
	}

	if ref != nil {
		return ref, nil
	}

	if err := repo.FetchContext(ctx, &git.FetchOptions{Depth: gf.options.Depth}); err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("fetching: %w", err)
	}

	ref, err = gf.find(repo, version)

	if err != nil {
		return nil, fmt.Errorf("finding version %q: %w", version, err)
	}

	return ref, nil
}

func (gf *GitFetcher) find(repo *git.Repository, version string) (*Reference, error) {
	if tag, err := repo.Tag(version); err == nil {
		annotatedTag, err := repo.TagObject(tag.Hash())

		// object not found is returned for lightweight tags
		if err != nil && !errors.Is(err, plumbing.ErrObjectNotFound) {
			return nil, fmt.Errorf("getting annotated tag: %w", err)
		}

		if annotatedTag != nil {
			return NewReferenceToAnnotatedTag(annotatedTag), nil
		}

		return NewReference(tag), nil
	}

	if branch, err := repo.Reference(plumbing.NewBranchReferenceName(version), true); err == nil {
		return NewReference(branch), nil
	}

	if partial, err := repo.Reference(plumbing.NewBranchReferenceName("refs/heads/"+version), true); err == nil {
		return NewReference(partial), nil
	}

	commits, err := repo.CommitObjects()
	if err != nil {
		return nil, fmt.Errorf("getting commit objects: %w", err)
	}
	defer commits.Close()

	var found *plumbing.Reference
	err = commits.ForEach(func(c *object.Commit) error {
		h := c.Hash.String()

		if h == version || (len(version) > 7 && strings.HasPrefix(h, version)) {
			found = plumbing.NewHashReference(plumbing.ReferenceName(h), c.ID())
			return storer.ErrStop
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("searching commits: %w", err)
	}

	if found == nil {
		return nil, nil
	}

	return NewReference(found), nil
}
