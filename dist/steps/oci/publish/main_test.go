package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/dist/steps/oci/publish/testutil/bldr"

	mainBldr "gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestRun(t *testing.T) {
	t.Run("publishes OCI step", func(t *testing.T) {
		registry := mainBldr.StartOCIRegistryServer(t)

		stepPath := mainBldr.Files(t).
			WriteFile("step.yml", "spec:\n---\nexec: {command: [sh]}").
			WriteFile("run", "1234").
			Build()

		outputFile := filepath.Join(t.TempDir(), "outputs.jsonl")
		cliArgs, getEnv := bldr.CLIInputs(t).
			WithRegistry(registry.Address()).
			WithRepository("my-image").
			WithTag("1.0.1").
			WithCommon(fmt.Sprintf(`{"files": {"%s": "step.yml"}}`, filepath.Join(stepPath, "step.yml"))).
			WithPlatforms(fmt.Sprintf(`{"linux/arm64": {"files": {"%s": "run"}}}`, filepath.Join(stepPath, "run"))).
			WithOutputFile(outputFile).
			Build()

		err := run(cliArgs, getEnv)
		require.NoError(t, err)

		outputs := extractOutputs(t, outputFile)
		require.Equal(t, registry.Address(), outputs["registry"])
		require.Equal(t, "my-image", outputs["repository"])
		require.Equal(t, "1.0.1", outputs["tag"])
		require.Equal(t, fmt.Sprintf("%s/my-image:1.0.1", registry.Address()), outputs["ref"])
		require.Contains(t, outputs["digest"], "sha256")
	})
}

func extractOutputs(t *testing.T, outputFile string) map[string]string {
	jsonl, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputs := make(map[string]string)

	for _, line := range strings.Split(strings.TrimSpace(string(jsonl)), "\n") {
		var m map[string]any

		err := json.Unmarshal([]byte(line), &m)
		require.NoError(t, err, line)

		if _, nameOk := m["name"]; !nameOk {
			require.Fail(t, "output jsonl line does not contain 'name'", line)
		}

		if _, valIsString := m["name"].(string); !valIsString {
			require.Fail(t, "output jsonl name is not a string", line)
		}

		if _, valueOk := m["value"]; !valueOk {
			require.Fail(t, "output jsonl line does not contain 'value'", line)
		}

		value := line
		if valueStr, ok := m["value"].(string); ok {
			value = valueStr
		}

		outputs[m["name"].(string)] = value
	}

	return outputs
}
