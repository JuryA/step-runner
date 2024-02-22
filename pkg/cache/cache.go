package cache

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Cache interface {
	Get(ctx context.Context, step *proto.Step_Reference) (*proto.StepDefinition, error)
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

func (c *cache) Get(ctx context.Context, stepRef *proto.Step_Reference) (*proto.StepDefinition, error) {
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
		return load(filepath.Join(stepRef.Path...))
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
	var err error
	k := cacheKey(step)
	dir := filepath.Join(c.cacheDir, string(k))
	fileInfo, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("reading cache for step %q (%v): %w", step, k, err)
	}
	if err == nil && !fileInfo.IsDir() {
		return "", fmt.Errorf("cache for step %q (%v) is not dir: %w", step, k, err)
	}
	if os.IsNotExist(err) {
		return c.cacheMiss(ctx, step, k)
	}
	return dir, nil
}

func (c *cache) cacheMiss(ctx context.Context, step *proto.Step_Reference, k key) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context cancelled: %w", err)
	}
	dir := filepath.Join(c.cacheDir, string(k))
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		return "", fmt.Errorf("making dir for cloning: %w", err)
	}
	err = execIn(dir, "git", "clone", step.Url, ".")
	if err != nil {
		return "", fmt.Errorf("cloning %q: %w", step.Url, err)
	}
	return dir, nil
}

func execIn(dir string, c string, args ...string) error {
	cmd := exec.Command(c, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v", err, string(out))
	}
	if cmd.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("exit code %v: %v", cmd.ProcessState.ExitCode(), string(out))
	}
	return nil
}

type key string

func cacheKey(step *proto.Step_Reference) key {
	k := step.Url
	// Temporarily reconstructing the original cache key. We are
	// going store steps in a directory structure:
	// https://gitlab.com/gitlab-org/step-runner/-/issues/5. But
	// that will be another MR so I'm just patching this back
	// together here to make the integration test pass.
	k = strings.Replace(k, "https://", "https+git://", 1)
	if step.Version != "" {
		k = k + "@" + step.Version
	}
	sum := sha256.Sum256([]byte(k))
	return key(fmt.Sprintf("%x", sum)[:8])
}
