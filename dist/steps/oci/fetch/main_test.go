package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/dist-steps/oci/fetch/internal/testutil/bldr"
)

func TestRun(t *testing.T) {
	t.Run("loads OCI step", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		layer := bldr.OCIImageLayer(t).WithFile("/step.yml", []byte("spec:\n---\nexec: {command: [bash]}")).Build()
		img := bldr.OCIImage(t).WithLayer(layer).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(remoteImgRef, imgIndex)

		outputFile := filepath.Join(t.TempDir(), "outputs.jsonl")
		cliArgs, getEnv := bldr.CLIInputs(t).WithRemoteImgRef(remoteImgRef).WithOutputFile(outputFile).Build()
		err := run(cliArgs, getEnv)
		require.NoError(t, err)

		outputs := extractOutputs(t, outputFile)
		require.Contains(t, outputs, "fetched_step_path")

		stepYml, err := os.ReadFile(outputs["fetched_step_path"])
		require.NoError(t, err)
		require.Equal(t, string(stepYml), "spec:\n---\nexec: {command: [bash]}")
	})

	t.Run("loads OCI step using a digest", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		layer := bldr.
			OCIImageLayer(t).
			WithFile("/step.yml", []byte("spec:\n---\nexec: {command: [sh]}")).
			Build()
		img := bldr.OCIImage(t).WithLayer(layer).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(registry.RefToImage("image", "latest"), imgIndex)

		digest, err := imgIndex.Digest()
		require.NoError(t, err)

		digestRef := registry.RefToImageDigest("image", digest)

		outputFile := filepath.Join(t.TempDir(), "outputs.jsonl")
		cliArgs, getEnv := bldr.CLIInputs(t).WithRemoteImgRef(digestRef).WithOutputFile(outputFile).Build()
		err = run(cliArgs, getEnv)
		require.NoError(t, err)

		outputs := extractOutputs(t, outputFile)
		require.Contains(t, outputs, "fetched_step_path")

		stepYml, err := os.ReadFile(outputs["fetched_step_path"])
		require.NoError(t, err)
		require.Equal(t, string(stepYml), "spec:\n---\nexec: {command: [sh]}")
	})

	t.Run("loads OCI step in sub-directory", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		layer := bldr.
			OCIImageLayer(t).
			WithFile("/foo/bar/bob/step.yml", []byte("spec:\n---\nexec: {command: [bash]}")).
			Build()
		img := bldr.OCIImage(t).WithLayer(layer).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(remoteImgRef, imgIndex)

		outputFile := filepath.Join(t.TempDir(), "outputs.jsonl")
		cliArgs, getEnv := bldr.CLIInputs(t).
			WithRemoteImgRef(remoteImgRef).
			WithOutputFile(outputFile).
			WithStepPath("foo/bar/bob").
			WithStepFile("step.yml").
			Build()
		err := run(cliArgs, getEnv)
		require.NoError(t, err)

		outputs := extractOutputs(t, outputFile)
		require.Contains(t, outputs, "fetched_step_path")

		stepYml, err := os.ReadFile(outputs["fetched_step_path"])
		require.NoError(t, err)
		require.Equal(t, string(stepYml), "spec:\n---\nexec: {command: [bash]}")
	})
}

func extractOutputs(t *testing.T, outputFile string) map[string]string {
	jsonl, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputs := make(map[string]string)

	for _, line := range strings.Split(strings.TrimSpace(string(jsonl)), "\n") {
		var m map[string]string

		err := json.Unmarshal([]byte(line), &m)
		require.NoError(t, err)

		if _, nameOk := m["name"]; !nameOk {
			t.Errorf("output jsonl line does not contain 'name': %s", line)
		}

		if _, valueOk := m["value"]; !valueOk {
			t.Errorf("output jsonl line does not contain 'value': %s", line)
		}

		outputs[m["name"]] = m["value"]
	}

	return outputs
}
