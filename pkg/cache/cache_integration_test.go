//go:build integration

package cache

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCacheRemote(t *testing.T) {
	key := "c6521ff4"

	// Test cache in temporary directory
	oldTempDir := os.Getenv("TMPDIR")
	tempDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer func() {
		os.RemoveAll(tempDir)
		os.Setenv("TMPDIR", oldTempDir)
	}()
	os.Setenv("TMPDIR", tempDir)
	_, err = os.Stat(filepath.Join(tempDir, "step-runner-cache"))
	require.True(t, os.IsNotExist(err))

	// Cache fetches exactly one step
	runEchoSteps(t)
	entries, err := os.ReadDir(filepath.Join(tempDir, "step-runner-cache"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, entries[0].Name(), key)
	require.FileExists(t, filepath.Join(tempDir, "step-runner-cache", key, "step.yml"))

	// Cache is reused
	runEchoSteps(t)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, entries[0].Name(), key)
	require.FileExists(t, filepath.Join(tempDir, "step-runner-cache", key, "step.yml"))
}

func runEchoSteps(t *testing.T) {
	t.Helper()
	cmd := exec.Command("go", "run", "../..", "ci")
	cmd.Env = append(os.Environ(), "STEPS="+echoSteps)
	out, err := cmd.CombinedOutput()
	require.Equal(t, 0, cmd.ProcessState.ExitCode(), string(out))
	require.NoError(t, err, string(out))
}

const echoSteps = `
- name: hello_world
  step: "https+git://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step"
  inputs:
    echo: hello world
`
