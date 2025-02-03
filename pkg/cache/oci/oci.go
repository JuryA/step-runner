package oci

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/mod/module"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal/client"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal/version"
)

var errNoLocalCache = errors.New("no local cache")

type Fetcher struct {
	cacheDir string
	mu       sync.Mutex

	client *client.Client
}

func New(cacheDir string) *Fetcher {
	return &Fetcher{
		cacheDir: cacheDir,
		client:   client.New(),
	}
}

func (f *Fetcher) Get(ctx context.Context, url, ver string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ref, err := client.ParseReference(url)
	if err != nil {
		return "", fmt.Errorf("parsing reference: %w", err)
	}

	// append version
	ref, err = name.NewTag(fmt.Sprintf("%s:%s", ref.Context().Name(), ver))
	if err != nil {
		return "", fmt.Errorf("reference tag: %w", err)
	}

	constraint, err := version.NewConstraint(ref.Identifier())
	if err != nil {
		return "", fmt.Errorf("parsing constraint: %w", err)
	}

	// if the version provided is not a constraint, then we check to see if we have this
	// version on disk first.
	if constraint.IsVersion() {
		dir, err := f.findLocal(ref)
		if err == nil {
			return dir, nil
		}

		if !errors.Is(err, errNoLocalCache) {
			return "", err
		}
	}

	// todo: for version constraints, we always try to pull, but we can probably
	// optimize this later.
	if err := f.client.Pull(ctx, ref.Name(), f.cacheDir); err != nil {
		return "", fmt.Errorf("pulling oci component: %w", err)
	}

	return f.findLocal(ref)
}

func (f *Fetcher) findLocal(ref name.Reference) (string, error) {
	constraint, err := version.NewConstraint(ref.Identifier())
	if err != nil {
		return "", fmt.Errorf("parsing constraint: %w", err)
	}

	dir, err := module.EscapePath(ref.Context().Name())
	if err != nil {
		return "", fmt.Errorf("escaping plugin path: %w", err)
	}
	dir = filepath.Join(f.cacheDir, dir)

	var versions []version.Version
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		v, err := module.UnescapeVersion(d.Name())
		if err != nil {
			return nil // ignore error
		}

		ver, err := version.New(v)
		if err != nil {
			return nil // ignore invalid versions
		}

		versions = append(versions, ver)

		return nil
	}); err != nil {
		return "", errNoLocalCache
	}

	versions = constraint.Match(versions)
	if len(versions) == 0 {
		return "", errNoLocalCache
	}

	selected := versions[len(versions)-1].String()
	selected, err = module.EscapeVersion(selected)
	if err != nil {
		return "", fmt.Errorf("escaping version: %w", err)
	}

	return filepath.Join(dir, selected), nil
}
