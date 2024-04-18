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
	repoParentDir := filepath.Join("gitlab.com", "gitlab-org", "ci-cd", "runner-tools")

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

	// Cache fetches the step
	runSteps(t, echoStepsMaster)
	entries, err := os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.FileExists(t, filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step@master", "step.yml"))

	// Cache separates by tag
	runSteps(t, echoStepsV1)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir))
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.FileExists(t, filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step@v1", "step.yml"))

	// Cache separates by hash
	runSteps(t, echoSteps91141a6e)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir))
	require.NoError(t, err)
	require.Len(t, entries, 3)
	require.FileExists(t, filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step@91141a6e", "step.yml"))

	// Cache supports nested steps
	runSteps(t, nestedEchoSteps)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir))
	require.NoError(t, err)
	require.Len(t, entries, 3) // will reuse cached echo-step@master
	require.FileExists(t, filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step@master", "another-echo", "another-step.yml"))

	// Cache is reused
	runSteps(t, echoStepsMaster)
	runSteps(t, echoStepsV1)
	runSteps(t, echoSteps91141a6e)
	runSteps(t, nestedEchoSteps)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir))
	require.NoError(t, err)
	require.Len(t, entries, 3)
}

func runSteps(t *testing.T, steps string) {
	t.Helper()
	cmd := exec.Command("go", "run", "../..", "ci")
	cmd.Env = append(os.Environ(), "STEPS="+steps)
	out, err := cmd.CombinedOutput()
	require.Equal(t, 0, cmd.ProcessState.ExitCode(), string(out))
	require.NoError(t, err, string(out))
}

const echoStepsMaster = `
- name: hello_world
  step: "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step@master"
  inputs:
    echo: hello world
`

const echoStepsV1 = `
- name: hello_world
  step: "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step@v1"
  inputs:
    echo: hello world
`

const echoSteps91141a6e = `
- name: hello_world
  step: "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step@91141a6e"
  inputs:
    echo: hello world
`

const nestedEchoSteps = `
- name: another_hello_world
  step:
    git:
      url: "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step"
      dir: another-echo/another-file.yml
      rev: master
  inputs:
    echo: hello other world
`
