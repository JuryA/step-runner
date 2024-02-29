package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Cache interface {
	Get(ctx context.Context, step *proto.Step_Reference) (*proto.StepDefinition, *proto.Step_Reference, error)
}

var _ Cache = &cache{}

type cache struct {
	mux      sync.Mutex
	cacheDir string
}

func New() (Cache, error) {
	cacheDir := filepath.Join(os.TempDir(), "step-runner-cache")
	err := os.MkdirAll(cacheDir, 0750)
	if err != nil {
		return nil, fmt.Errorf("making cache dir %q: %w", cacheDir, err)
	}
	return &cache{
		cacheDir: cacheDir,
	}, nil
}

func (c *cache) Get(ctx context.Context, stepRef *proto.Step_Reference) (*proto.StepDefinition, *proto.Step_Reference, error) {
	load := func(dir string) (*proto.StepDefinition, error) {
		filename := filepath.Join(dir, "step.yml")
		stepDef, err := step.LoadSteps(filename)
		if err != nil {
			return nil, fmt.Errorf("loading file %q: %w", dir, err)
		}
		protoStepDef, err := step.CompileSteps(stepDef)
		if err != nil {
			return nil, fmt.Errorf("compiling file %q: %w", dir, err)
		}
		return protoStepDef, nil
	}
	switch {
	case stepRef.Protocol == proto.StepReferenceProtocol_local:
		stepDef, err := load(filepath.Join(stepRef.Path...))
		// We don't rewrite local references
		return stepDef, stepRef, err
	case stepRef.Protocol == proto.StepReferenceProtocol_git:
		dir, cacheStepRef, err := c.getCacheDir(ctx, stepRef)
		if err != nil {
			return nil, nil, fmt.Errorf("fetching step %q: %w", stepRef, err)
		}
		stepDef, err := load(dir)
		return stepDef, cacheStepRef, err
	default:
		return nil, nil, fmt.Errorf("invalid step reference: %v", stepRef)
	}
}

func (c *cache) getCacheDir(ctx context.Context, step *proto.Step_Reference) (string, *proto.Step_Reference, error) {
	c.mux.Lock()
	defer c.mux.Unlock()
	_, after, found := strings.Cut(step.Url, "//")
	if !found {
		return "", nil, fmt.Errorf("invalid step url. expected '//': %q", step.Url)
	}
	repoPath := strings.Split(after, "/")
	if len(repoPath) == 0 {
		return "", nil, fmt.Errorf("missing repo path after '//'")
	}
	if step.Version != "" {
		// Append version to differentiate versions of the same repo
		last := repoPath[len(repoPath)-1]
		last += "@" + step.Version
		repoPath[len(repoPath)-1] = last
	}
	for i, d := range repoPath {
		e, err := escapeString(d)
		if err != nil {
			return "", nil, fmt.Errorf("escaping path: %w", err)
		}
		repoPath[i] = e
	}
	dir := filepath.Join(c.cacheDir, filepath.Join(repoPath...))
	fileInfo, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", nil, fmt.Errorf("reading cache for step %q (%v): %w", step, step, err)
	}
	if err == nil && !fileInfo.IsDir() {
		return "", nil, fmt.Errorf("cache for step %q (%v) is not dir: %w", step, step, err)
	}
	if os.IsNotExist(err) {
		if err := c.cacheMiss(ctx, step, dir); err != nil {
			return "", nil, err
		}
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", nil, fmt.Errorf("opening repo: %w", err)
	}
	head, err := repo.Head()
	if err != nil {
		return "", nil, fmt.Errorf("reading head: %w", err)
	}
	hash := head.Hash().String()
	if len(hash) < 8 {
		return "", nil, fmt.Errorf("invalid hash %q", hash)
	}
	hash = hash[:8]
	cacheStepRef := &proto.Step_Reference{
		Protocol: step.Protocol,
		Url:      step.Url,
		Path:     step.Path,
		Filename: step.Filename,
		Version:  hash,
	}
	return dir, cacheStepRef, nil
}

func (c *cache) cacheMiss(ctx context.Context, step *proto.Step_Reference, dir string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		return fmt.Errorf("making dir for cloning: %w", err)
	}
	if regexp.MustCompile("[A-Fa-f0-9]{8}").Match([]byte(step.Version)) {
		return cloneHash(step.Url, step.Version, dir)
	} else {
		return cloneTag(step.Url, step.Version, dir)
	}
}

func cloneTag(url, tag, dir string) error {
	_, err := git.PlainClone(dir, false, &git.CloneOptions{
		Depth:             1,
		RecurseSubmodules: git.SubmoduleRescursivity(1),
		ReferenceName:     plumbing.ReferenceName("refs/tags/" + tag),
		SingleBranch:      true,
		URL:               url,
	})
	if err != nil {
		return fmt.Errorf("cloning %q: %w", url, err)
	}
	return nil
}

func cloneHash(url, hash, dir string) error {
	repo, err := git.PlainClone(dir, false, &git.CloneOptions{
		Depth:             1,
		RecurseSubmodules: git.SubmoduleRescursivity(1),
		SingleBranch:      true,
		URL:               url,
	})
	if err != nil {
		return fmt.Errorf("cloning %q: %w", url, err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting workgree: %w", err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{}); err != nil {
		return fmt.Errorf("checking out %q: %w", hash, err)
	}
	return nil
}

// Forked from https://cs.opensource.google/go/x/mod/+/refs/tags/v0.15.0:module/module.go
func escapeString(s string) (escaped string, err error) {
	haveUpper := false
	for _, r := range s {
		if r == '!' || r >= utf8.RuneSelf {
			return "", fmt.Errorf("internal error: inconsistency in escapeString")
		}
		if 'A' <= r && r <= 'Z' {
			haveUpper = true
		}
	}

	if !haveUpper {
		return s, nil
	}

	var buf []byte
	for _, r := range s {
		if 'A' <= r && r <= 'Z' {
			buf = append(buf, '!', byte(r+'a'-'A'))
		} else {
			buf = append(buf, byte(r))
		}
	}
	return string(buf), nil
}
