package integration_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestCanBuildAndFetchAnImage(t *testing.T) {
	registry := bldr.StartOCIRegistryServer(t, bldr.WithRequireAuth("fred", "c0d3_n1nj4"))

	baseDir := bldr.Files(t).
		WriteFile("step.yml", "spec:\n---\nexec:\n  command: [cat, '${{step_dir}}/app/files/templates/message']").
		WriteFile("files/templates_dir/message", "Hello, World!").
		Build()

	template := `
spec:
---
run:
  - name: build_image
    step: dist://step/oci/build
    inputs:
      registry: %s
      repository: my-image
      tag: pipeline-111234
      common:
        files:
          %s/step.yml: step.yml
      platforms:
        %s/%s:  
          files:
            %s/files/templates_dir: /app/files/templates
  - name: run_built_image
    step:
      oci:
        registry: %s
        repository: my-image
        tag: pipeline-111234
  - name: echo_registry
    script: "echo reg: ${{steps.build_image.outputs.registry}}"
  - name: echo_repository
    script: "echo repo: ${{steps.build_image.outputs.repository}}"
  - name: echo_tag
    script: "echo tag: ${{steps.build_image.outputs.tag}}"
  - name: echo_ref
    script: "echo ref: ${{steps.build_image.outputs.ref}}"
  - name: echo_algorithm
    script: "echo algorithm: ${{steps.build_image.outputs.digest.algorithm}}"
  - name: echo_hash
    script: "echo hash: ${{steps.build_image.outputs.digest.hash}}"
  - name: echo_digest
    script: "echo digest: ${{steps.build_image.outputs.digest.value}}"`

	platform := bldr.OCIPlatform.ThisPlatform
	registryAddr := registry.Address()
	testStep := fmt.Sprintf(template, registryAddr, baseDir, platform.OS, platform.Architecture, baseDir, registryAddr)
	userPass := base64.StdEncoding.EncodeToString([]byte("fred:c0d3_n1nj4"))

	_, logs, err := testutil.StepRunner(t).
		WithDebugLogs().
		WithEnvKeyVal("DOCKER_AUTH_CONFIG", fmt.Sprintf(`{"auths":{"%s":{"auth":"%s"}}}`, registryAddr, userPass)).
		Run(testStep)
	require.NoError(t, err)
	require.Contains(t, logs, "Hello, World!")
	require.Contains(t, logs, `Running step "build_image"`)
	require.Regexp(t, `INFO fetched step image=.*/my-image:pipeline-111234`, logs)
	require.Contains(t, logs, `Running step "run_built_image"`)
	require.Contains(t, logs, "Hello, World!")
	require.Contains(t, logs, "reg: "+registryAddr)
	require.Contains(t, logs, "repo: my-image")
	require.Contains(t, logs, "tag: pipeline-111234")
	require.Regexp(t, `ref: .*/my-image:pipeline-111234`, logs)
	require.Contains(t, logs, "algorithm: sha256")
	require.Regexp(t, `hash: [0-9a-f]{64}`, logs)
	require.Regexp(t, `digest: sha256:[0-9a-f]{64}`, logs)
}
