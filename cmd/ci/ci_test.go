package ci

import (
	"os"
	"testing"

	"gitlab.com/gitlab-org/step-runner/proto"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCICmd(t *testing.T) {
	stepResultsFile := "step-results.json"
	t.Run("runs steps", func(t *testing.T) {
		steps := `
- name: secret_factory_a
  step: ../../pkg/runner/test_steps/secret_factory
- name: secret_factory_b
  step: ../../pkg/runner/test_steps/secret_factory
  inputs:
    secret_override: ${{ steps.secret_factory_a.outputs.secret }}
`
		require.NoError(t, os.Setenv("STEPS", steps))
		defer func() { _ = os.Unsetenv("STEPS") }()

		cmd := NewCmd()
		cmd.SetArgs([]string{"--write-steps-results"})
		err := cmd.Execute()
		require.NoError(t, err)

		file, err := os.ReadFile(stepResultsFile)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, result.Status)

		require.Len(t, result.SubStepResults, 2)
		require.Equal(t, result.SubStepResults[0].Outputs["secret"], result.SubStepResults[1].Outputs["secret"])
	})

	t.Run("generates step-results file", func(t *testing.T) {
		tests := []struct {
			name             string
			env              map[string]string
			args             []string
			expectFileExists bool
		}{
			{
				name:             "generates step file when CLI arg used",
				args:             []string{"--write-steps-results"},
				expectFileExists: true,
			},
			{
				name:             "generates step file when env variable set",
				env:              map[string]string{"CI_STEPS_DEBUG": "true"},
				expectFileExists: true,
			},
			{
				name:             "does not generate step file when env variable not set and CLI arg not used",
				env:              map[string]string{},
				args:             []string{},
				expectFileExists: false,
			},
			{
				name:             "does not generate step file when env variable set to false",
				env:              map[string]string{"CI_STEPS_DEBUG": "false"},
				expectFileExists: false,
			},
			{
				name:             "does not generate step file when env variable not valid bool",
				env:              map[string]string{"CI_STEPS_DEBUG": "invalid.bool"},
				expectFileExists: false,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				_ = os.Remove(stepResultsFile)

				require.NoError(t, os.Setenv("STEPS", `- step: ../../pkg/runner/test_steps/secret_factory`))
				defer func() { _ = os.Unsetenv("STEPS") }()

				for key, value := range test.env {
					defer func() { _ = os.Unsetenv(key) }()
					require.NoError(t, os.Setenv(key, value))
				}

				cmd := NewCmd()
				cmd.SetArgs(test.args)
				err := cmd.Execute()
				require.NoError(t, err)

				if test.expectFileExists {
					require.FileExists(t, stepResultsFile)
				} else {
					require.NoFileExists(t, stepResultsFile)
				}
			})
		}
	})

	t.Run("failed step returns error", func(t *testing.T) {
		steps := `
- name: exit
  step: ../../pkg/runner/test_steps/exit
  inputs:
    exit_code: 99
`
		require.NoError(t, os.Setenv("STEPS", steps))
		defer func() { _ = os.Unsetenv("STEPS") }()

		cmd := NewCmd()
		cmd.SetArgs([]string{"--write-steps-results"})
		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "exec: exit status 99")

		file, err := os.ReadFile(stepResultsFile)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_failure, result.Status)
	})
}
