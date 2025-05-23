package run

import (
	"fmt"
	"os"
	"testing"

	"gitlab.com/gitlab-org/step-runner/pkg/report"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/proto"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
)

const (
	stepResultsFileJSON      = "step-results.json"
	stepResultsFileProtoText = "step-results.txtpb"
)

func TestRunCmd(t *testing.T) {
	t.Run("runs step loaded from file", func(t *testing.T) {
		_ = os.Remove(stepResultsFileJSON)

		cmd := NewCmd()
		cmd.SetArgs([]string{
			"--write-steps-results",
			"../../integration_test/steps/secret_factory",
			"--inputs",
			"secret_override=secrety.secret",
			"--env",
			"FOO=BAR",
			"--step-results-file",
			stepResultsFileJSON,
		})
		err := cmd.Execute()
		require.NoError(t, err)

		file, err := os.ReadFile(stepResultsFileJSON)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, result.Status)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "secrety.secret", result.SubStepResults[0].Outputs["secret"].GetStringValue())
		require.Equal(t, "BAR", result.SubStepResults[0].Env["FOO"])
	})

	t.Run("runs step loaded using YAML syntax", func(t *testing.T) {
		_ = os.Remove(stepResultsFileJSON)

		cmd := NewCmd()
		cmd.SetArgs([]string{
			"step: ../../integration_test/steps/exit",
			"--inputs",
			"exit_code=0",
			"--write-steps-results",
			"--step-results-file",
			stepResultsFileJSON,
		})
		err := cmd.Execute()
		require.NoError(t, err)

		file, err := os.ReadFile(stepResultsFileJSON)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, result.Status)
	})

	t.Run("failed step returns error", func(t *testing.T) {
		_ = os.Remove(stepResultsFileJSON)

		cmd := NewCmd()
		cmd.SetArgs([]string{
			"../../integration_test/steps/exit",
			"--inputs",
			"exit_code=99",
			"--write-steps-results",
			"--step-results-file",
			stepResultsFileJSON,
		})
		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "exec: exit status 99")

		file, err := os.ReadFile(stepResultsFileJSON)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_failure, result.Status)
	})

	t.Run("can accept prototext input and produce prototext step result file", func(t *testing.T) {
		_ = os.Remove(stepResultsFileProtoText)

		cmd := NewCmd()
		cmd.SetArgs([]string{
			"--text-proto-step-file",
			"hello_world.txtpb",
			"--step-results-format",
			string(report.FormatProtoText),
			"--step-results-file",
			stepResultsFileProtoText,
		})
		err := cmd.Execute()
		require.NoError(t, err)

		require.NoError(t, err)
		data, err := os.ReadFile(stepResultsFileProtoText)
		require.NoError(t, err)

		result := &proto.StepResult{}
		err = prototext.Unmarshal(data, result)
		require.NoError(t, err)
		require.Len(t, result.SubStepResults, 2)
		require.Contains(t, result.SubStepResults[1].Outputs, "echo")
		require.Equal(t, "hello world", result.SubStepResults[1].Outputs["echo"].GetStringValue(), string(data))
	})

	t.Run("runs step loaded from OCI image", func(t *testing.T) {
		_ = os.Remove(stepResultsFileJSON)

		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		echoStepContents, err := os.ReadFile("../../integration_test/steps/echo/step.yml")
		require.NoError(t, err)

		img := bldr.OCIImage(t).WithFile("/step.yml", echoStepContents).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(remoteImgRef, imgIndex)

		cmd := NewCmd()
		cmd.SetArgs([]string{
			"--write-steps-results",
			"--oci-registry", registry.Address(),
			"--oci-repository", remoteImgRef.Context().RepositoryStr(),
			"--oci-tag", remoteImgRef.Identifier(),
			"--inputs",
			"echo=hello world",
			"--step-results-file",
			stepResultsFileJSON,
		})
		err = cmd.Execute()
		require.NoError(t, err)

		file, err := os.ReadFile(stepResultsFileJSON)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, result.Status)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "hello world", result.SubStepResults[0].Outputs["echo"].GetStringValue(), string(file))
	})

	t.Run("runs step loaded from OCI image in subdirectory with different filename", func(t *testing.T) {
		_ = os.Remove(stepResultsFileJSON)

		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		stepYAML := `
spec:
  outputs:
    echo:
      type: string
---
exec:
  command: [ sh, -c, "cat ${{step_dir}}/foo.json >${{output_file}}" ]`

		fooJSON := fmt.Sprintf("%s\n", `{"name": "echo", "value": "hello world!"}`)

		img := bldr.OCIImage(t).
			WithFile("/foo/bar/foo-step.yml", []byte(stepYAML)).
			WithFile("/foo/bar/foo.json", []byte(fooJSON)).
			Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(remoteImgRef, imgIndex)

		cmd := NewCmd()
		cmd.SetArgs([]string{
			"--write-steps-results",
			"--oci-registry", registry.Address(),
			"--oci-repository", remoteImgRef.Context().RepositoryStr(),
			"--oci-tag", remoteImgRef.Identifier(),
			"--oci-dir", "/foo/bar",
			"--oci-filename", "foo-step.yml",
			"--step-results-file",
			stepResultsFileJSON,
		})
		err := cmd.Execute()
		require.NoError(t, err)

		file, err := os.ReadFile(stepResultsFileJSON)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, result.Status)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "hello world!", result.SubStepResults[0].Outputs["echo"].GetStringValue(), string(file))
	})
}
