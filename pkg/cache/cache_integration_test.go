//go:build integration

package cache

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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

	// Cache fetches the step by branch
	runSteps(t, echoStepsBranch)
	entries, err := os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step/91141a6e0d41411744703f9a5aa4417502263842"))
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.FileExists(t, filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step/91141a6e0d41411744703f9a5aa4417502263842", "step.yml"))

	// Cache re-uses hash of branch via tag v1
	runSteps(t, echoStepsV1)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step/91141a6e0d41411744703f9a5aa4417502263842"))
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.FileExists(t, filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step/91141a6e0d41411744703f9a5aa4417502263842", "step.yml"))

	// Cache fetches v2 tag
	runSteps(t, echoStepsV2)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step/6b81e62af5b8c1a856aeda4259136caa6a029c39"))
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.FileExists(t, filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step/6b81e62af5b8c1a856aeda4259136caa6a029c39", "step.yml"))

	// Cache fetches via specific commit
	runSteps(t, echoSteps91141a6e)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step/91141a6e0d41411744703f9a5aa4417502263842"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.FileExists(t, filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step/91141a6e0d41411744703f9a5aa4417502263842", "step.yml"))

	// Cache supports nested steps
	runSteps(t, nestedEchoSteps)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.FileExists(t, filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step/702d1554b550be67407516e1595c24dcba78e8b2", "another-echo", "another-step.yml"))

	// Cache is reused
	runSteps(t, echoStepsBranch)
	runSteps(t, echoStepsV1)
	runSteps(t, echoSteps91141a6e)
	runSteps(t, nestedEchoSteps)
	runSteps(t, echoStepsV2)
	entries, err = os.ReadDir(filepath.Join(tempDir, "step-runner-cache", repoParentDir, "echo-step"))
	require.NoError(t, err)
	require.Len(t, entries, 4)
}

func runSteps(t *testing.T, steps string) {
	t.Helper()
	cmd := exec.Command("go", "run", "../..", "ci")
	cmd.Env = append(os.Environ(), "STEPS="+steps)
	out, err := cmd.CombinedOutput()
	require.Equal(t, 0, cmd.ProcessState.ExitCode(), string(out))
	require.NoError(t, err, string(out))
}

const echoStepsBranch = `
- name: hello_world
  step: "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step@points-to-91141a6e"
  inputs:
    echo: hello world
`

const echoStepsV1 = `
- name: hello_world
  step: "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step@v1"
  inputs:
    echo: hello world
`

const echoStepsV2 = `
- name: hello_world
  step: "https://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step@v2"
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
      dir: reverse
      rev: 702d1554
  inputs:
    echo: hello world in reverse
`
