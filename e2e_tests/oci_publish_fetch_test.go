package e2e_tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/testutil"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestCanFetchAPublishedImage(t *testing.T) {
	registry := bldr.StartOCIRegistryServer(t)

	baseDir := bldr.Files(t).
		WriteFile("step.yml", `spec: --- exec: { command: [cat, ${{step_dir}}/app/files/templates/message] }`).
		WriteFile("files/templates_dir/message", "Hello, World!").
		Build()

	template := `
spec:
---
run:
  - name: publish_image
    step: "builtin://oci/publish"
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
`

	platform := bldr.OCIPlatform.ThisPlatform
	testStep := fmt.Sprintf(template, registry.Address(), baseDir, platform.OS, platform.Architecture, baseDir)

	_, err := testutil.StepRunner(t).Run(testStep)
	require.NoError(t, err)
}
