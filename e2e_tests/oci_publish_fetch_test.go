package e2e_tests

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestCanFetchAPublishedImage(t *testing.T) {
	registry := bldr.StartOCIRegistryServer(t)

	baseDir := bldr.Files(t).
		WriteFile("step.yml", "spec:\n---\nexec:\n  command: [cat, '${{step_dir}}/app/files/templates/message']").
		WriteFile("files/templates_dir/message", "Hello, World!").
		Build()

	template := `
spec:
---
run:
  - name: publish_image
    step: builtin://oci/publish
    inputs:
      registry: %s
      repository: my-image
      tag: 1.0.2
      common:
        files:
          %s/step.yml: step.yml
      platforms:
        %s_%s:  
          files:
            %s/files/templates_dir: /app/files/templates
  - name: run_published_step
    step:
      oci:
        registry: %s
        repository: my-image
        tag: 1.0.2`

	platform := bldr.OCIPlatform.ThisPlatform
	registryAddr := registry.Address()
	testStep := fmt.Sprintf(template, registryAddr, baseDir, platform.OS, platform.Architecture, baseDir, registryAddr)

	buffer := &bytes.Buffer{}
	_, err := testutil.StepRunner(t).WithLogs(buffer).Run(testStep)
	require.NoError(t, err)
	require.Contains(t, buffer.String(), "Hello, World!")
}
