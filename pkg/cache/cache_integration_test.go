// //go:build integration

package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestCacheRemote(t *testing.T) {
	oldTempDir := os.Getenv("TMPDIR")
	tempDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer func() {
		os.RemoveAll(tempDir)
		os.Setenv("TMPDIR", oldTempDir)
	}()
	os.Setenv("TMPDIR", tempDir)
	c, err := New()
	runnerToolsDir := filepath.Join(c.(*cache).cacheDir, "gitlab.com", "gitlab-org", "ci-cd", "runner-tools")

	sequence := []struct {
		ref           *proto.Step_Reference
		wantDir       string
		wantVersion   string
		wantCacheSize int
	}{{
		ref: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step",
			Path:     nil,
			Filename: "step.yml",
			Version:  "v2",
		},
		wantDir:       filepath.Join(runnerToolsDir, "echo-step@v2"),
		wantVersion:   "6b81e62a", // rewritten from v2 tag
		wantCacheSize: 1,
	}, {
		ref: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step",
			Path:     nil,
			Filename: "step.yml",
			Version:  "91141a6e",
		},
		wantDir:       filepath.Join(runnerToolsDir, "echo-step@91141a6e"),
		wantVersion:   "91141a6e",
		wantCacheSize: 2,
	}, {
		ref: &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step",
			Path:     []string{"another-echo"},
			Filename: "another-step.yml",
			Version:  "v2",
		},
		wantDir:       filepath.Join(runnerToolsDir, "echo-step@v2", "another-step"),
		wantVersion:   "91141a6e", // rewritten from v2 tag
		wantCacheSize: 2,
	}}

	for _, s := range sequence {
		_, gotStepRef, err := c.Get(context.Background(), s.ref)
		require.NoError(t, err)
		require.Equal(t, s.wantVersion, gotStepRef.Version)
		entries, err := os.ReadDir(s.wantDir)
		require.NoError(t, err)
		require.Len(t, entries, 2)
		require.FileExists(t, filepath.Join(s.wantDir, s.ref.Filename))
	}
}
