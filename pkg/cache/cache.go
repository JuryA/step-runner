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
	Get(ctx context.Context, step string) (*proto.StepDefinition, error)
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

func (c *cache) Get(ctx context.Context, stepRef string) (*proto.StepDefinition, error) {
	load := func(dir string) (*proto.StepDefinition, error) {
		filename := filepath.Join(dir, "step.yml")
		stepDefinition, err := step.Read(filename)
		if err != nil {
			return nil, fmt.Errorf("loading file %q: %w", dir, err)
		}
		return stepDefinition, nil
	}
	switch {
	case strings.HasPrefix(stepRef, "."):
		return load(stepRef)
	case strings.HasPrefix(stepRef, "https+git"):
		dir, err := c.getCacheDir(ctx, stepRef)
		if err != nil {
			return nil, fmt.Errorf("fetching step %q: %w", stepRef, err)
		}
		return load(dir)
	default:
		return nil, fmt.Errorf("invalid step reference: %v", stepRef)
	}
}

func (c *cache) getCacheDir(ctx context.Context, step string) (string, error) {
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

func (c *cache) cacheMiss(ctx context.Context, step string, k key) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context cancelled: %w", err)
	}
	dir := filepath.Join(c.cacheDir, string(k))
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		return "", fmt.Errorf("making dir for cloning: %w", err)
	}
	url := strings.Replace(step, "https+git", "https", 1)
	err = execIn(dir, "git", "clone", url, ".")
	if err != nil {
		return "", fmt.Errorf("cloning %q: %w", url, err)
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

func cacheKey(uri string) key {
	sum := sha256.Sum256([]byte(uri))
	return key(fmt.Sprintf("%x", sum)[:8])
}
