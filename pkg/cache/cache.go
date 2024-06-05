package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/go-git/go-git/v5"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Cache interface {
	Get(ctx context.Context, parentDir string, step *proto.Step_Reference) (*proto.SpecDefinition, error)
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

func (c *cache) Get(ctx context.Context, parentDir string, stepRef *proto.Step_Reference) (*proto.SpecDefinition, error) {
	load := func(dir string) (*proto.SpecDefinition, error) {
		path := filepath.Join(stepRef.Path...)
		filename := filepath.Join(dir, path, stepRef.Filename)
		stepDef, err := step.LoadSteps(filename)
		if err != nil {
			return nil, fmt.Errorf("loading file %q: %w", filename, err)
		}
		protoStepDef, err := step.CompileSteps(stepDef)
		if err != nil {
			return nil, fmt.Errorf("compiling file %q: %w", dir, err)
		}
		protoStepDef.Dir = filepath.Join(dir, path)
		return protoStepDef, nil
	}
	switch {
	case stepRef.Protocol == proto.StepReferenceProtocol_local:
		return load(parentDir)
	case stepRef.Protocol == proto.StepReferenceProtocol_git:
		dir, err := c.getCacheDir(ctx, stepRef)
		if err != nil {
			return nil, fmt.Errorf("fetching step %q: %w", stepRef, err)
		}
		return load(dir)
	default:
		return nil, fmt.Errorf("invalid step reference: %v", stepRef)
	}
}

func (c *cache) getCacheDir(ctx context.Context, step *proto.Step_Reference) (string, error) {
	c.mux.Lock()
	defer c.mux.Unlock()
	_, after, found := strings.Cut(step.Url, "//")
	if !found {
		return "", fmt.Errorf("invalid step url. expected '//': %q", step.Url)
	}
	repoPath := strings.Split(after, "/")
	if len(repoPath) == 0 {
		return "", fmt.Errorf("missing repo path after '//'")
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
			return "", fmt.Errorf("escaping path: %w", err)
		}
		repoPath[i] = e
	}
	dir := filepath.Join(c.cacheDir, filepath.Join(repoPath...))
	fileInfo, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("reading cache for step %q (%v): %w", step, step, err)
	}
	if err == nil && !fileInfo.IsDir() {
		return "", fmt.Errorf("cache for step %q (%v) is not dir: %w", step, step, err)
	}
	if os.IsNotExist(err) {
		return c.cacheMiss(ctx, step, dir)
	}
	return dir, nil
}

func (c *cache) cacheMiss(ctx context.Context, step *proto.Step_Reference, dir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context cancelled: %w", err)
	}
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		return "", fmt.Errorf("making dir for cloning: %w", err)
	}
	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		Depth:             1,
		SingleBranch:      true,
		RecurseSubmodules: git.SubmoduleRescursivity(1),
		URL:               step.Url,
	})
	if err != nil {
		return "", fmt.Errorf("cloning %q: %w", step.Url, err)
	}
	return dir, nil
}

// Forked from https://cs.opensource.google/go/x/mod/+/refs/tags/v0.15.0:module/module.go
func escapeString(s string) (escaped string, err error) {
	haveUpper := false
	for _, r := range s {
		if r == '!' || r >= utf8.RuneSelf {
			// This should be disallowed by CheckPath, but diagnose anyway.
			// The correctness of the escaping loop below depends on it.
			return "", fmt.Errorf("internal error: inconsistency in EscapePath")
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
